package memory

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

type fakeMemoryProvider struct {
	base  uintptr
	data  []byte
	calls int
	reads []fakeMemoryRead
	err   error
}

type fakeMemoryRead struct {
	address uintptr
	size    int
}

func (f *fakeMemoryProvider) ReadAt(address uintptr, dst []byte) error {
	f.calls++
	f.reads = append(f.reads, fakeMemoryRead{address: address, size: len(dst)})
	if f.err != nil {
		return f.err
	}
	if address < f.base {
		return errors.New("read before fake memory base")
	}
	offset := int(address - f.base)
	if offset+len(dst) > len(f.data) {
		return errors.New("read beyond fake memory")
	}
	copy(dst, f.data[offset:offset+len(dst)])
	return nil
}

func (f *fakeMemoryProvider) Stats() ReadStats {
	return ReadStats{Calls: uint64(f.calls)}
}

func TestReadStringBoundedStopsAtTerminator(t *testing.T) {
	provider := &fakeMemoryProvider{
		base: 0x1000,
		data: []byte("hello\x00ignored"),
	}
	process := &Process{provider: provider}

	got, err := process.ReadStringBounded(0x1000, 0, uint(len(provider.data)))
	if err != nil {
		t.Fatalf("ReadStringBounded returned error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("ReadStringBounded() = %q, want %q", got, "hello")
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
}

func TestReadBatchCoalescesNearbyRequests(t *testing.T) {
	provider := &fakeMemoryProvider{
		base: 0x2000,
		data: make([]byte, 256),
	}
	for i := range provider.data {
		provider.data[i] = byte(i)
	}
	process := &Process{provider: provider}

	first := make([]byte, 4)
	second := make([]byte, 4)
	results := process.ReadBatch([]ReadRequest{
		{Name: "first", Address: 0x2004, Dst: first},
		{Name: "second", Address: 0x200c, Dst: second},
	})

	for _, result := range results {
		if result.Err != nil {
			t.Fatalf("ReadBatch result %q returned error: %v", result.Name, result.Err)
		}
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
	if !bytes.Equal(first, []byte{4, 5, 6, 7}) {
		t.Fatalf("first buffer = %v, want [4 5 6 7]", first)
	}
	if !bytes.Equal(second, []byte{12, 13, 14, 15}) {
		t.Fatalf("second buffer = %v, want [12 13 14 15]", second)
	}
}

func TestReadBatchSeparatesDistantRequests(t *testing.T) {
	provider := &fakeMemoryProvider{
		base: 0x3000,
		data: make([]byte, 512),
	}
	process := &Process{provider: provider}

	first := make([]byte, 4)
	second := make([]byte, 4)
	process.ReadBatch([]ReadRequest{
		{Name: "first", Address: 0x3004, Dst: first},
		{Name: "second", Address: 0x30c8, Dst: second},
	})

	if provider.calls != 2 {
		t.Fatalf("provider calls = %d, want 2", provider.calls)
	}
}

func TestDiffReadStats(t *testing.T) {
	before := ReadStats{Calls: 5, Bytes: 100, Failures: 1}
	after := ReadStats{Calls: 9, Bytes: 164, Failures: 2, LastLatency: 3 * time.Millisecond}

	got := diffReadStats(after, before)
	if got.Calls != 4 || got.Bytes != 64 || got.Failures != 1 || got.LastLatency != after.LastLatency {
		t.Fatalf("diffReadStats() = %+v", got)
	}
}
