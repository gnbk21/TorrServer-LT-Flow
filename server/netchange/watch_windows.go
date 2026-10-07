//go:build windows

// Package netchange provides coalesced notification hints. Polling remains the
// source of truth; callbacks never perform networking or wait on application locks.
package netchange

import (
	"golang.org/x/sys/windows"
	"sync"
	"sync/atomic"
	"unsafe"
)

var subscribers sync.Map
var nextID atomic.Uint64

// Go callback trampolines cannot be freed. Allocate one for the process, not
// three more on every engine reconnect.
var callback = windows.NewCallback(func(context, _, _ uintptr) uintptr {
	if v, ok := subscribers.Load(context); ok {
		select {
		case v.(chan struct{}) <- struct{}{}:
		default:
		}
	}
	return 0
})

func Watch() (<-chan struct{}, func()) {
	events := make(chan struct{}, 1)
	id := uintptr(nextID.Add(1))
	subscribers.Store(id, events)
	dll := windows.NewLazySystemDLL("iphlpapi.dll")
	var handles []windows.Handle
	for _, name := range []string{"NotifyIpInterfaceChange", "NotifyRouteChange2", "NotifyUnicastIpAddressChange"} {
		var handle windows.Handle
		r, _, _ := dll.NewProc(name).Call(0, callback, id, 0, uintptr(unsafe.Pointer(&handle)))
		if r == 0 && handle != 0 {
			handles = append(handles, handle)
		}
	}
	var once sync.Once
	return events, func() {
		once.Do(func() {
			subscribers.Delete(id)
			for _, h := range handles {
				dll.NewProc("CancelMibChangeNotify2").Call(uintptr(h))
			}
		})
	}
}
