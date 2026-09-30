//go:build windows

package diagnostics

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

var processMemoryInfo = windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo")
var processHandleCount = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetProcessHandleCount")

func processMemory() (uint64, uint64, bool, bool) {
	var counters struct {
		Size, PageFaultCount                                                                                                                       uint32
		PeakWorkingSet, WorkingSet, QuotaPeakPagedPool, QuotaPagedPool, QuotaPeakNonPagedPool, QuotaNonPagedPool, PagefileUsage, PeakPagefileUsage uintptr
	}
	counters.Size = uint32(unsafe.Sizeof(counters))
	ok, _, _ := processMemoryInfo.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&counters)), uintptr(counters.Size))
	var handles uint32
	hOK, _, _ := processHandleCount.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&handles)))
	return uint64(counters.WorkingSet), uint64(handles), ok != 0, hOK != 0
}
