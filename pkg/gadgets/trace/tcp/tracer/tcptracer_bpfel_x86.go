// Code generated by bpf2go; DO NOT EDIT.
//go:build 386 || amd64
// +build 386 amd64

package tracer

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"

	"github.com/cilium/ebpf"
)

// loadTcptracer returns the embedded CollectionSpec for tcptracer.
func loadTcptracer() (*ebpf.CollectionSpec, error) {
	reader := bytes.NewReader(_TcptracerBytes)
	spec, err := ebpf.LoadCollectionSpecFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("can't load tcptracer: %w", err)
	}

	return spec, err
}

// loadTcptracerObjects loads tcptracer and converts it into a struct.
//
// The following types are suitable as obj argument:
//
//     *tcptracerObjects
//     *tcptracerPrograms
//     *tcptracerMaps
//
// See ebpf.CollectionSpec.LoadAndAssign documentation for details.
func loadTcptracerObjects(obj interface{}, opts *ebpf.CollectionOptions) error {
	spec, err := loadTcptracer()
	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, opts)
}

// tcptracerSpecs contains maps and programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type tcptracerSpecs struct {
	tcptracerProgramSpecs
	tcptracerMapSpecs
}

// tcptracerSpecs contains programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type tcptracerProgramSpecs struct {
	IgTcpAccept *ebpf.ProgramSpec `ebpf:"ig_tcp_accept"`
	IgTcpClose  *ebpf.ProgramSpec `ebpf:"ig_tcp_close"`
	IgTcpState  *ebpf.ProgramSpec `ebpf:"ig_tcp_state"`
	IgTcpV4CoE  *ebpf.ProgramSpec `ebpf:"ig_tcp_v4_co_e"`
	IgTcpV4CoX  *ebpf.ProgramSpec `ebpf:"ig_tcp_v4_co_x"`
	IgTcpV6CoE  *ebpf.ProgramSpec `ebpf:"ig_tcp_v6_co_e"`
	IgTcpV6CoX  *ebpf.ProgramSpec `ebpf:"ig_tcp_v6_co_x"`
}

// tcptracerMapSpecs contains maps before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type tcptracerMapSpecs struct {
	Events        *ebpf.MapSpec `ebpf:"events"`
	MountNsFilter *ebpf.MapSpec `ebpf:"mount_ns_filter"`
	Sockets       *ebpf.MapSpec `ebpf:"sockets"`
	Tuplepid      *ebpf.MapSpec `ebpf:"tuplepid"`
}

// tcptracerObjects contains all objects after they have been loaded into the kernel.
//
// It can be passed to loadTcptracerObjects or ebpf.CollectionSpec.LoadAndAssign.
type tcptracerObjects struct {
	tcptracerPrograms
	tcptracerMaps
}

func (o *tcptracerObjects) Close() error {
	return _TcptracerClose(
		&o.tcptracerPrograms,
		&o.tcptracerMaps,
	)
}

// tcptracerMaps contains all maps after they have been loaded into the kernel.
//
// It can be passed to loadTcptracerObjects or ebpf.CollectionSpec.LoadAndAssign.
type tcptracerMaps struct {
	Events        *ebpf.Map `ebpf:"events"`
	MountNsFilter *ebpf.Map `ebpf:"mount_ns_filter"`
	Sockets       *ebpf.Map `ebpf:"sockets"`
	Tuplepid      *ebpf.Map `ebpf:"tuplepid"`
}

func (m *tcptracerMaps) Close() error {
	return _TcptracerClose(
		m.Events,
		m.MountNsFilter,
		m.Sockets,
		m.Tuplepid,
	)
}

// tcptracerPrograms contains all programs after they have been loaded into the kernel.
//
// It can be passed to loadTcptracerObjects or ebpf.CollectionSpec.LoadAndAssign.
type tcptracerPrograms struct {
	IgTcpAccept *ebpf.Program `ebpf:"ig_tcp_accept"`
	IgTcpClose  *ebpf.Program `ebpf:"ig_tcp_close"`
	IgTcpState  *ebpf.Program `ebpf:"ig_tcp_state"`
	IgTcpV4CoE  *ebpf.Program `ebpf:"ig_tcp_v4_co_e"`
	IgTcpV4CoX  *ebpf.Program `ebpf:"ig_tcp_v4_co_x"`
	IgTcpV6CoE  *ebpf.Program `ebpf:"ig_tcp_v6_co_e"`
	IgTcpV6CoX  *ebpf.Program `ebpf:"ig_tcp_v6_co_x"`
}

func (p *tcptracerPrograms) Close() error {
	return _TcptracerClose(
		p.IgTcpAccept,
		p.IgTcpClose,
		p.IgTcpState,
		p.IgTcpV4CoE,
		p.IgTcpV4CoX,
		p.IgTcpV6CoE,
		p.IgTcpV6CoX,
	)
}

func _TcptracerClose(closers ...io.Closer) error {
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Do not access this directly.
//go:embed tcptracer_bpfel_x86.o
var _TcptracerBytes []byte
