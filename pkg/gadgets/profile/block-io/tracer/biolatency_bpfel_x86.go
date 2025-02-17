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

type biolatencyHist struct{ Slots [27]uint32 }

type biolatencyHistKey struct {
	CmdFlags uint32
	Dev      uint32
}

// loadBiolatency returns the embedded CollectionSpec for biolatency.
func loadBiolatency() (*ebpf.CollectionSpec, error) {
	reader := bytes.NewReader(_BiolatencyBytes)
	spec, err := ebpf.LoadCollectionSpecFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("can't load biolatency: %w", err)
	}

	return spec, err
}

// loadBiolatencyObjects loads biolatency and converts it into a struct.
//
// The following types are suitable as obj argument:
//
//     *biolatencyObjects
//     *biolatencyPrograms
//     *biolatencyMaps
//
// See ebpf.CollectionSpec.LoadAndAssign documentation for details.
func loadBiolatencyObjects(obj interface{}, opts *ebpf.CollectionOptions) error {
	spec, err := loadBiolatency()
	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, opts)
}

// biolatencySpecs contains maps and programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type biolatencySpecs struct {
	biolatencyProgramSpecs
	biolatencyMapSpecs
}

// biolatencySpecs contains programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type biolatencyProgramSpecs struct {
	IgProfioDone *ebpf.ProgramSpec `ebpf:"ig_profio_done"`
	IgProfioIns  *ebpf.ProgramSpec `ebpf:"ig_profio_ins"`
	IgProfioIss  *ebpf.ProgramSpec `ebpf:"ig_profio_iss"`
}

// biolatencyMapSpecs contains maps before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type biolatencyMapSpecs struct {
	CgroupMap *ebpf.MapSpec `ebpf:"cgroup_map"`
	Hists     *ebpf.MapSpec `ebpf:"hists"`
	Start     *ebpf.MapSpec `ebpf:"start"`
}

// biolatencyObjects contains all objects after they have been loaded into the kernel.
//
// It can be passed to loadBiolatencyObjects or ebpf.CollectionSpec.LoadAndAssign.
type biolatencyObjects struct {
	biolatencyPrograms
	biolatencyMaps
}

func (o *biolatencyObjects) Close() error {
	return _BiolatencyClose(
		&o.biolatencyPrograms,
		&o.biolatencyMaps,
	)
}

// biolatencyMaps contains all maps after they have been loaded into the kernel.
//
// It can be passed to loadBiolatencyObjects or ebpf.CollectionSpec.LoadAndAssign.
type biolatencyMaps struct {
	CgroupMap *ebpf.Map `ebpf:"cgroup_map"`
	Hists     *ebpf.Map `ebpf:"hists"`
	Start     *ebpf.Map `ebpf:"start"`
}

func (m *biolatencyMaps) Close() error {
	return _BiolatencyClose(
		m.CgroupMap,
		m.Hists,
		m.Start,
	)
}

// biolatencyPrograms contains all programs after they have been loaded into the kernel.
//
// It can be passed to loadBiolatencyObjects or ebpf.CollectionSpec.LoadAndAssign.
type biolatencyPrograms struct {
	IgProfioDone *ebpf.Program `ebpf:"ig_profio_done"`
	IgProfioIns  *ebpf.Program `ebpf:"ig_profio_ins"`
	IgProfioIss  *ebpf.Program `ebpf:"ig_profio_iss"`
}

func (p *biolatencyPrograms) Close() error {
	return _BiolatencyClose(
		p.IgProfioDone,
		p.IgProfioIns,
		p.IgProfioIss,
	)
}

func _BiolatencyClose(closers ...io.Closer) error {
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Do not access this directly.
//go:embed biolatency_bpfel_x86.o
var _BiolatencyBytes []byte
