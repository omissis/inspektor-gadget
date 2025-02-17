//go:build linux
// +build linux

// Copyright 2019-2022 The Inspektor Gadget authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tracer

// #include <linux/types.h>
// #include "./bpf/tcptracer.h"
// #include <arpa/inet.h>
// #include <stdlib.h>
//
//static char *addr_str(const void *addr, __u32 af) {
//	size_t size = af == AF_INET ? INET_ADDRSTRLEN : INET6_ADDRSTRLEN;
//	char *str;
//
//	str = malloc(size);
//	if (!str)
//		return NULL;
//
//	inet_ntop(af, addr, str, size);
//
//	return str;
//}
//
//static char *get_src_addr(const struct event *ev) {
//	if (ev->af == AF_INET)
//		return addr_str(&ev->saddr_v4, ev->af);
//	else if (ev->af == AF_INET6)
//		return addr_str(&ev->saddr_v6, ev->af);
//	else
//		return NULL;
//}
//
//static char *get_dst_addr(const struct event *ev) {
//	if (ev->af == AF_INET)
//		return addr_str(&ev->daddr_v4, ev->af);
//	else if (ev->af == AF_INET6)
//		return addr_str(&ev->daddr_v6, ev->af);
//	else
//		return NULL;
//}
import "C"

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/tcp/types"
	eventtypes "github.com/inspektor-gadget/inspektor-gadget/pkg/types"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -target $TARGET -cc clang -no-global-types tcptracer ./bpf/tcptracer.bpf.c -- -I./bpf/ -I../../../../${TARGET}

type Config struct {
	MountnsMap *ebpf.Map
}

type Tracer struct {
	config        *Config
	enricher      gadgets.DataEnricher
	eventCallback func(types.Event)

	objs tcptracerObjects

	tcpv4connectEnterLink link.Link
	tcpv4connectExitLink  link.Link
	tcpv6connectEnterLink link.Link
	tcpv6connectExitLink  link.Link
	tcpCloseEnterLink     link.Link
	tcpSetStateEnterLink  link.Link
	inetCskAcceptExitLink link.Link

	reader *perf.Reader
}

func NewTracer(config *Config, enricher gadgets.DataEnricher,
	eventCallback func(types.Event),
) (*Tracer, error) {
	t := &Tracer{
		config:        config,
		enricher:      enricher,
		eventCallback: eventCallback,
	}

	if err := t.start(); err != nil {
		t.Stop()
		return nil, err
	}

	return t, nil
}

func (t *Tracer) Stop() {
	t.tcpv4connectEnterLink = gadgets.CloseLink(t.tcpv4connectEnterLink)
	t.tcpv4connectExitLink = gadgets.CloseLink(t.tcpv4connectExitLink)
	t.tcpv6connectEnterLink = gadgets.CloseLink(t.tcpv6connectEnterLink)
	t.tcpv6connectExitLink = gadgets.CloseLink(t.tcpv6connectExitLink)
	t.tcpCloseEnterLink = gadgets.CloseLink(t.tcpCloseEnterLink)
	t.tcpSetStateEnterLink = gadgets.CloseLink(t.tcpSetStateEnterLink)
	t.inetCskAcceptExitLink = gadgets.CloseLink(t.inetCskAcceptExitLink)

	if t.reader != nil {
		t.reader.Close()
	}

	t.objs.Close()
}

func (t *Tracer) start() error {
	spec, err := loadTcptracer()
	if err != nil {
		return fmt.Errorf("failed to load ebpf program: %w", err)
	}

	mapReplacements := map[string]*ebpf.Map{}
	filterByMntNs := false

	if t.config.MountnsMap != nil {
		filterByMntNs = true
		mapReplacements["mount_ns_filter"] = t.config.MountnsMap
	}

	consts := map[string]interface{}{
		"filter_by_mnt_ns": filterByMntNs,
	}

	if err := spec.RewriteConstants(consts); err != nil {
		return fmt.Errorf("error RewriteConstants: %w", err)
	}

	opts := ebpf.CollectionOptions{
		MapReplacements: mapReplacements,
	}

	if err := spec.LoadAndAssign(&t.objs, &opts); err != nil {
		return fmt.Errorf("failed to load ebpf program: %w", err)
	}

	t.tcpv4connectEnterLink, err = link.Kprobe("tcp_v4_connect", t.objs.IgTcpV4CoE, nil)
	if err != nil {
		return fmt.Errorf("error opening kprobe: %w", err)
	}

	t.tcpv4connectExitLink, err = link.Kretprobe("tcp_v4_connect", t.objs.IgTcpV4CoX, nil)
	if err != nil {
		return fmt.Errorf("error opening kprobe: %w", err)
	}

	t.tcpv6connectEnterLink, err = link.Kprobe("tcp_v6_connect", t.objs.IgTcpV6CoE, nil)
	if err != nil {
		return fmt.Errorf("error opening kprobe: %w", err)
	}

	t.tcpv6connectExitLink, err = link.Kretprobe("tcp_v6_connect", t.objs.IgTcpV6CoX, nil)
	if err != nil {
		return fmt.Errorf("error opening kprobe: %w", err)
	}

	// TODO: rename function in ebpf program
	t.tcpCloseEnterLink, err = link.Kprobe("tcp_close", t.objs.IgTcpClose, nil)
	if err != nil {
		return fmt.Errorf("error opening kprobe: %w", err)
	}

	t.tcpSetStateEnterLink, err = link.Kprobe("tcp_set_state", t.objs.IgTcpState, nil)
	if err != nil {
		return fmt.Errorf("error opening kprobe: %w", err)
	}

	t.inetCskAcceptExitLink, err = link.Kretprobe("inet_csk_accept", t.objs.IgTcpAccept, nil)
	if err != nil {
		return fmt.Errorf("error opening kprobe: %w", err)
	}

	reader, err := perf.NewReader(t.objs.tcptracerMaps.Events, gadgets.PerfBufferPages*os.Getpagesize())
	if err != nil {
		return fmt.Errorf("error creating perf ring buffer: %w", err)
	}
	t.reader = reader

	go t.run()

	return nil
}

func (t *Tracer) run() {
	for {
		record, err := t.reader.Read()
		if err != nil {
			if errors.Is(err, perf.ErrClosed) {
				// nothing to do, we're done
				return
			}

			msg := fmt.Sprintf("Error reading perf ring buffer: %s", err)
			t.eventCallback(types.Base(eventtypes.Err(msg)))
			return
		}

		if record.LostSamples > 0 {
			msg := fmt.Sprintf("lost %d samples", record.LostSamples)
			t.eventCallback(types.Base(eventtypes.Warn(msg)))
			continue
		}

		eventC := (*C.struct_event)(unsafe.Pointer(&record.RawSample[0]))

		event := types.Event{
			Event: eventtypes.Event{
				Type: eventtypes.NORMAL,
			},
			MountNsID: uint64(eventC.mntns_id),
			Pid:       uint32(eventC.pid),
			Comm:      C.GoString(&eventC.task[0]),
			Dport:     uint16(C.htons(eventC.dport)),
			Sport:     uint16(C.htons(eventC.sport)),
		}

		if eventC.af == C.AF_INET {
			event.IPVersion = 4
		} else if eventC.af == C.AF_INET6 {
			event.IPVersion = 6
		}

		if eventC._type == C.TCP_EVENT_TYPE_CONNECT {
			event.Operation = "connect"
		} else if eventC._type == C.TCP_EVENT_TYPE_ACCEPT {
			event.Operation = "accept"
		} else if eventC._type == C.TCP_EVENT_TYPE_CLOSE {
			event.Operation = "close"
		}

		srcAddr := C.get_src_addr(eventC)
		event.Saddr = C.GoString(srcAddr)
		C.free(unsafe.Pointer(srcAddr))

		dstAddr := C.get_dst_addr(eventC)
		event.Daddr = C.GoString(dstAddr)
		C.free(unsafe.Pointer(dstAddr))

		if t.enricher != nil {
			t.enricher.Enrich(&event.CommonData, event.MountNsID)
		}

		t.eventCallback(event)
	}
}
