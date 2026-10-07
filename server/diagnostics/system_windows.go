//go:build windows

package diagnostics

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

func systemMemory() (uint64, uint64, bool) {
	var s struct {
		Length, Load                                                                     uint32
		TotalPhys, AvailPhys, TotalPage, AvailPage, TotalVirtual, AvailVirtual, Extended uint64
	}
	s.Length = uint32(unsafe.Sizeof(s))
	r, _, _ := windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx").Call(uintptr(unsafe.Pointer(&s)))
	return s.TotalPhys, s.AvailPhys, r != 0
}
func DiskFree(path string) (uint64, bool) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, false
	}
	var free, total, available uint64
	err = windows.GetDiskFreeSpaceEx(p, &free, &total, &available)
	return free, err == nil
}
