package lt

/*
#include "lt_shim.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"unsafe"
)

// SetStreamFrontiers publishes immediate reader misses and active bootstrap
// blockers. Native evidence expires after three seconds; no new window is made.
func (t *Torrent) SetStreamFrontiers(pieces []int) error {
	if t == nil || len(pieces) > 8 {
		return ErrInvalid
	}
	if pieces == nil {
		pieces = []int{}
	}
	for _, piece := range pieces {
		if piece < 0 {
			return ErrInvalid
		}
	}
	data, err := json.Marshal(pieces)
	if err != nil {
		return err
	}
	value := C.CString(string(data))
	defer C.free(unsafe.Pointer(value))
	return codeToErr(C.lt_torrent_stream_frontiers(t.id, value))
}
