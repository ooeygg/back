package memory

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	moduleName                  = "d2r.exe"
	defaultMaxRemoteStringBytes = uint(4096)
	remoteStringChunkSize       = uint(256)
	readCoalesceGap             = uintptr(64)
)

type Process struct {
	handler              windows.Handle
	pid                  uint32
	moduleBaseAddressPtr uintptr
	moduleBaseSize       uint32
	provider             MemoryProvider
	sendPacket           *sendPacketState
	sendPacketMu         sync.Mutex
}

// MemoryProvider is the process-memory read boundary. Parsers should depend on
// Process methods, and Process delegates actual remote reads through this
// provider so read metrics and alternative implementations stay centralized.
type MemoryProvider interface {
	ReadAt(address uintptr, dst []byte) error
	Stats() ReadStats
}

// ReadStats contains coarse counters for remote memory reads made through a
// MemoryProvider.
type ReadStats struct {
	Calls       uint64
	Bytes       uint64
	Failures    uint64
	LastLatency time.Duration
}

type readStatsCounters struct {
	calls            atomic.Uint64
	bytes            atomic.Uint64
	failures         atomic.Uint64
	lastLatencyNanos atomic.Int64
}

func (s *readStatsCounters) record(size int, latency time.Duration, err error) {
	s.calls.Add(1)
	s.bytes.Add(uint64(size))
	s.lastLatencyNanos.Store(latency.Nanoseconds())
	if err != nil {
		s.failures.Add(1)
	}
}

func (s *readStatsCounters) snapshot() ReadStats {
	return ReadStats{
		Calls:       s.calls.Load(),
		Bytes:       s.bytes.Load(),
		Failures:    s.failures.Load(),
		LastLatency: time.Duration(s.lastLatencyNanos.Load()),
	}
}

type win32MemoryProvider struct {
	handle windows.Handle
	stats  readStatsCounters
}

func newWin32MemoryProvider(handle windows.Handle) *win32MemoryProvider {
	return &win32MemoryProvider{handle: handle}
}

func (p *win32MemoryProvider) ReadAt(address uintptr, dst []byte) error {
	if len(dst) == 0 {
		return nil
	}
	start := time.Now()
	err := windows.ReadProcessMemory(p.handle, address, &dst[0], uintptr(len(dst)), nil)
	p.stats.record(len(dst), time.Since(start), err)
	return err
}

func (p *win32MemoryProvider) Stats() ReadStats {
	return p.stats.snapshot()
}

// ReadRequest is a single remote memory read destination used by ReadBatch.
type ReadRequest struct {
	Name    string
	Address uintptr
	Dst     []byte
}

// ReadResult reports the outcome of one ReadBatch request.
type ReadResult struct {
	Name    string
	Address uintptr
	Size    int
	Err     error
}

type readRange struct {
	base uintptr
	end  uintptr
	reqs []int
}

const (
	Int8  = 1
	Int16 = 2
	Int32 = 4
	Int64 = 8
)

func NewProcess() (*Process, error) {
	module, err := getGameModule()
	if err != nil {
		return nil, err
	}

	h, err := windows.OpenProcess(0x0010, false, module.ProcessID)
	if err != nil {
		return nil, err
	}

	return &Process{
		handler:              h,
		pid:                  module.ProcessID,
		moduleBaseAddressPtr: module.ModuleBaseAddress,
		moduleBaseSize:       module.ModuleBaseSize,
		provider:             newWin32MemoryProvider(h),
	}, nil
}

func NewProcessForPID(pid uint32) (*Process, error) {
	module, found := getMainModule(pid)
	if !found {
		return nil, errors.New("no module found for the specified PID")
	}

	h, err := windows.OpenProcess(0x0010, false, module.ProcessID)
	if err != nil {
		return nil, err
	}

	return &Process{
		handler:              h,
		pid:                  module.ProcessID,
		moduleBaseAddressPtr: module.ModuleBaseAddress,
		moduleBaseSize:       module.ModuleBaseSize,
		provider:             newWin32MemoryProvider(h),
	}, nil
}

func (p *Process) Close() error {
	if p == nil || p.handler == 0 {
		return nil
	}
	return windows.CloseHandle(p.handler)
}

// ReadStats returns remote-read counters for this process.
func (p *Process) ReadStats() ReadStats {
	if p == nil || p.provider == nil {
		return ReadStats{}
	}
	return p.provider.Stats()
}

func (p *Process) readAt(address uintptr, dst []byte) error {
	if len(dst) == 0 {
		return nil
	}
	if p == nil {
		return errors.New("process is nil")
	}
	if p.provider != nil {
		return p.provider.ReadAt(address, dst)
	}
	if p.handler == 0 {
		return errors.New("process handle is invalid")
	}
	return windows.ReadProcessMemory(p.handler, address, &dst[0], uintptr(len(dst)), nil)
}

// ModuleBaseAddress returns the base address of the D2R module.
func (p *Process) ModuleBaseAddress() uintptr {
	return p.moduleBaseAddressPtr
}

func getGameModule() (ModuleInfo, error) {
	processes := make([]uint32, 2048)
	length := uint32(0)
	err := windows.EnumProcesses(processes, &length)
	if err != nil {
		return ModuleInfo{}, err
	}

	for _, process := range processes {
		module, found := getMainModule(process)
		if found {
			return module, nil
		}
	}

	return ModuleInfo{}, err
}

func getMainModule(pid uint32) (ModuleInfo, bool) {
	mi, err := GetProcessModules(pid)
	if err != nil {
		return ModuleInfo{}, false
	}
	for _, m := range mi {
		if strings.Contains(strings.ToLower(m.ModuleName), moduleName) {
			return m, true
		}
	}

	return ModuleInfo{}, false
}

func (p *Process) getProcessMemory() ([]byte, error) {
	// Use chunked reading as primary method since VirtualQueryEx is often blocked
	return p.getProcessMemoryChunked()
}

// getProcessMemoryChunked reads memory in small chunks, skipping protected regions
func (p *Process) getProcessMemoryChunked() ([]byte, error) {
	return ReadMemoryChunked(p.handler, p.moduleBaseAddressPtr, p.moduleBaseSize)
}

// ReadMemoryChunked reads memory in small chunks, skipping protected regions
// This is useful for reading large modules where a single ReadProcessMemory call may fail
func ReadMemoryChunked(handle windows.Handle, baseAddress uintptr, size uint32) ([]byte, error) {
	var data = make([]byte, size)
	const pageSize = uintptr(4096)

	successfulReads := 0
	failedReads := 0

	for offset := uintptr(0); offset < uintptr(size); offset += pageSize {
		address := baseAddress + offset
		chunkSize := pageSize

		// Adjust last chunk
		if offset+pageSize > uintptr(size) {
			chunkSize = uintptr(size) - offset
		}

		// Try to read, but don't fail if it's protected
		err := windows.ReadProcessMemory(handle, address, &data[offset], chunkSize, nil)
		if err != nil {
			failedReads++
			// Fill with zeros and continue (pattern matching will fail gracefully)
			for i := offset; i < offset+chunkSize; i++ {
				data[i] = 0
			}
		} else {
			successfulReads++
		}
	}

	return data, nil
}

type IntType uint

const (
	Uint8  = 1
	Uint16 = 2
	Uint32 = 4
	Uint64 = 8
)

// ReadBytes reads exactly size bytes from remote process memory.
func (p *Process) ReadBytes(address uintptr, size uint) ([]byte, error) {
	if size == 0 {
		return []byte{}, nil
	}
	data := make([]byte, size)
	if err := p.readAt(address, data); err != nil {
		return data, err
	}
	return data, nil
}

func (p *Process) ReadBytesFromMemory(address uintptr, size uint) []byte {
	data, _ := p.ReadBytes(address, size)
	return data
}

// ReadUIntWithError reads an unsigned little-endian integer from remote memory.
func (p *Process) ReadUIntWithError(address uintptr, size IntType) (uint, error) {
	if size != Uint8 && size != Uint16 && size != Uint32 && size != Uint64 {
		return 0, errors.New("invalid unsigned integer read size")
	}
	data, err := p.ReadBytes(address, uint(size))
	if err != nil {
		return 0, err
	}
	return bytesToUint(data, size), nil
}

func (p *Process) ReadUInt(address uintptr, size IntType) uint {
	value, _ := p.ReadUIntWithError(address, size)
	return value
}

// ReadBatch reads multiple memory ranges and coalesces adjacent requests into
// fewer underlying process-memory reads.
func (p *Process) ReadBatch(reqs []ReadRequest) []ReadResult {
	results := make([]ReadResult, len(reqs))
	for i, req := range reqs {
		results[i] = ReadResult{Name: req.Name, Address: req.Address, Size: len(req.Dst)}
	}
	if len(reqs) == 0 {
		return results
	}

	for _, rr := range coalesceReadRequests(reqs) {
		buffer := make([]byte, int(rr.end-rr.base))
		if err := p.ReadIntoBuffer(rr.base, buffer); err != nil {
			for _, idx := range rr.reqs {
				results[idx].Err = err
			}
			continue
		}

		for _, idx := range rr.reqs {
			req := reqs[idx]
			offset := int(req.Address - rr.base)
			copy(req.Dst, buffer[offset:offset+len(req.Dst)])
		}
	}

	return results
}

func coalesceReadRequests(reqs []ReadRequest) []readRange {
	order := make([]int, 0, len(reqs))
	for i, req := range reqs {
		if len(req.Dst) > 0 {
			order = append(order, i)
		}
	}
	sort.Slice(order, func(i, j int) bool {
		return reqs[order[i]].Address < reqs[order[j]].Address
	})

	ranges := make([]readRange, 0, len(order))
	for _, idx := range order {
		req := reqs[idx]
		base := req.Address
		end := base + uintptr(len(req.Dst))

		if len(ranges) == 0 || base > ranges[len(ranges)-1].end+readCoalesceGap {
			ranges = append(ranges, readRange{base: base, end: end, reqs: []int{idx}})
			continue
		}

		last := &ranges[len(ranges)-1]
		if end > last.end {
			last.end = end
		}
		last.reqs = append(last.reqs, idx)
	}

	return ranges
}

func ReadUIntFromBuffer(buf []byte, offset uint, size IntType) uint {
	return bytesToUint(buf[offset:offset+uint(size)], size)
}

func bytesToUint(buf []byte, size IntType) uint {
	switch size {
	case Uint8:
		return uint(buf[0])
	case Uint16:
		return uint(binary.LittleEndian.Uint16(buf))
	case Uint32:
		return uint(binary.LittleEndian.Uint32(buf))
	case Uint64:
		return uint(binary.LittleEndian.Uint64(buf))
	}

	return 0
}
func ReadIntFromBuffer(buf []byte, offset uint, size IntType) int {
	return bytesToInt(buf[offset:offset+uint(size)], size)
}
func bytesToInt(buf []byte, size IntType) int {
	switch size {
	case Int8:
		return int(int8(buf[0]))
	case Int16:
		return int(int16(binary.LittleEndian.Uint16(buf)))
	case Int32:
		return int(int32(binary.LittleEndian.Uint32(buf)))
	case Int64:
		return int(int64(binary.LittleEndian.Uint64(buf)))
	}
	return 0
}

// ReadStringBounded reads a NUL-terminated remote string with a hard maximum
// length. When size is nonzero, it reads that fixed width and trims NUL bytes.
func (p *Process) ReadStringBounded(address uintptr, size, max uint) (string, error) {
	if address == 0 {
		return "", errors.New("cannot read string at null address")
	}
	if max == 0 {
		max = defaultMaxRemoteStringBytes
	}
	if size > max {
		size = max
	}
	if size > 0 {
		data, err := p.ReadBytes(address, size)
		if err != nil {
			return "", err
		}
		return string(bytes.Trim(data, "\x00")), nil
	}

	out := make([]byte, 0, remoteStringChunkSize)
	for read := uint(0); read < max; {
		chunkSize := remoteStringChunkSize
		if remaining := max - read; remaining < chunkSize {
			chunkSize = remaining
		}

		chunk := make([]byte, chunkSize)
		if err := p.ReadIntoBuffer(address+uintptr(read), chunk); err != nil {
			return string(bytes.Trim(out, "\x00")), err
		}

		if idx := bytes.IndexByte(chunk, 0); idx >= 0 {
			out = append(out, chunk[:idx]...)
			return string(out), nil
		}
		out = append(out, chunk...)
		read += chunkSize
	}

	return string(bytes.Trim(out, "\x00")), nil
}

func (p *Process) ReadStringFromMemory(address uintptr, size uint) string {
	value, _ := p.ReadStringBounded(address, size, defaultMaxRemoteStringBytes)
	return value
}

func (p *Process) findPattern(memory []byte, pattern, mask string) int {
	patternLength := len(pattern)
	for i := 0; i < int(p.moduleBaseSize)-patternLength; i++ {
		found := true
		for j := 0; j < patternLength; j++ {
			if string(mask[j]) != "?" && string(pattern[j]) != string(memory[i+j]) {
				found = false
				break
			}
		}

		if found {
			return i
		}
	}

	return 0
}

func (p *Process) FindPattern(memory []byte, pattern, mask string) uintptr {
	if offset := p.findPattern(memory, pattern, mask); offset != 0 {
		return p.moduleBaseAddressPtr + uintptr(offset)
	}

	return 0
}

func (p *Process) FindPatternByOperand(memory []byte, pattern, mask string) uintptr {
	if offset := p.findPattern(memory, pattern, mask); offset != 0 {
		// Adjust the address based on the operand value
		operandAddress := p.moduleBaseAddressPtr + uintptr(offset)
		operandValue := binary.LittleEndian.Uint32(memory[offset+3 : offset+7])
		finalAddress := operandAddress + uintptr(operandValue) + 7 // 7 is the length of the instruction
		return finalAddress
	}

	return 0
}

func (p *Process) GetPID() uint32 {
	return p.pid
}

type ModuleInfo struct {
	ProcessID         uint32
	ModuleBaseAddress uintptr
	ModuleBaseSize    uint32
	ModuleHandle      syscall.Handle
	ModuleName        string
}

func GetProcessModules(processID uint32) ([]ModuleInfo, error) {
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, processID)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(hProcess)

	var modules [1024]windows.Handle
	var needed uint32
	if err := windows.EnumProcessModules(hProcess, &modules[0], uint32(unsafe.Sizeof(modules[0]))*1024, &needed); err != nil {
		return nil, err
	}
	count := needed / uint32(unsafe.Sizeof(modules[0]))

	var moduleInfos []ModuleInfo
	for i := uint32(0); i < count; i++ {
		var mi windows.ModuleInfo
		if err := windows.GetModuleInformation(hProcess, modules[i], &mi, uint32(unsafe.Sizeof(mi))); err != nil {
			return nil, err
		}

		var moduleName [windows.MAX_PATH]uint16
		if err := windows.GetModuleFileNameEx(hProcess, modules[i], &moduleName[0], windows.MAX_PATH); err != nil {
			return nil, err
		}

		moduleInfos = append(moduleInfos, ModuleInfo{
			ProcessID:         processID,
			ModuleBaseAddress: mi.BaseOfDll,
			ModuleBaseSize:    mi.SizeOfImage,
			ModuleHandle:      syscall.Handle(modules[i]),
			ModuleName:        syscall.UTF16ToString(moduleName[:]),
		})
	}

	return moduleInfos, nil
}

// ReadPointer reads a pointer from the specified memory address.
func (p *Process) ReadPointer(address uintptr, size int) (uintptr, error) {
	if size != Int8 && size != Int16 && size != Int32 && size != Int64 {
		return 0, errors.New("invalid pointer read size")
	}
	value, err := p.ReadUIntWithError(address, IntType(size))
	return uintptr(value), err
}

func (p *Process) ReadIntoBuffer(address uintptr, buffer []byte) error {
	return p.readAt(address, buffer)
}

// ReadWidgetContainer reads the WidgetContainer structure.
func (p *Process) ReadWidgetContainer(address uintptr, full bool) (map[string]interface{}, error) {
	widgetPtr, err := p.ReadPointer(address+0x8, 8)
	if err != nil {
		return nil, err
	}

	widgetNameLength := p.ReadUInt(address+0x10, 4)

	widgetName := p.ReadStringFromMemory(widgetPtr, uint(widgetNameLength))
	if widgetName == "" {
		return nil, errors.New("failed to read widget name")
	}

	widget_visible := p.ReadUInt(address+0x51, 1) == 1
	widget_active := p.ReadUInt(address+0x50, 1) == 1

	result := map[string]interface{}{
		"WidgetNameString": widgetName,
		"WidgetNameLength": widgetNameLength,
		"WidgetVisible":    widget_visible,
		"WidgetActive":     widget_active,
	}

	if full {
		childWidgetsListPtr, err := p.ReadPointer(widgetPtr+0x38, 8)
		if err != nil {
			return nil, err
		}

		childWidgetSize := p.ReadUInt(widgetPtr+0x40, 4)

		widgetListPtr, err := p.ReadPointer(widgetPtr+0x68, 8)
		if err != nil {
			return nil, err
		}

		widgetListSize := p.ReadUInt(widgetPtr+0x78, 4)

		widgetList2Ptr, err := p.ReadPointer(widgetPtr+0x80, 8)
		if err != nil {
			return nil, err
		}

		widgetList2Size := p.ReadUInt(widgetPtr+0x90, 4)

		result["ChildWidgetsListPointer"] = childWidgetsListPtr
		result["ChildWidgetSize"] = childWidgetSize
		result["WidgetListPointer"] = widgetListPtr
		result["WidgetListSize"] = widgetListSize
		result["WidgetList2Pointer"] = widgetList2Ptr
		result["WidgetList2Size"] = widgetList2Size
	}

	return result, nil
}

// ReadWidgetList iterates through a list of widgets given a pointer to the list and its size.
func (p *Process) ReadWidgetList(listPointer uintptr, listSize int) (map[string]map[string]interface{}, error) {
	widgetMap := make(map[string]map[string]interface{})
	widgetSize := int(unsafe.Sizeof(uintptr(0)))

	for i := 0; i < listSize; i++ {
		widgetAddr, err := p.ReadPointer(listPointer+uintptr(i*widgetSize), 8)
		if err != nil {
			return nil, err
		}

		widgetContainer, err := p.ReadWidgetContainer(widgetAddr, false)
		if err != nil {
			return nil, err
		}

		widgetName, ok := widgetContainer["WidgetNameString"].(string)
		if !ok {
			return nil, errors.New("failed to read widget name")
		}

		widgetMap[widgetName] = widgetContainer
	}

	return widgetMap, nil
}
