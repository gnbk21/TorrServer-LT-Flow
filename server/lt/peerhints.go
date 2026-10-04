package lt

/*
#include "lt_shim.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"server/flow"
	"unsafe"
)

func (t *Torrent) ResumePeerHints() ([]flow.PeerHint, error) {
	data, err := cAlloc(func(n *C.size_t) *C.char { return C.lt_torrent_resume_peers_json_alloc(t.id, n) })
	if err != nil {
		return nil, err
	}
	var peers []flow.PeerHint
	err = json.Unmarshal(data, &peers)
	return peers, err
}

func (t *Torrent) RestorePeerHints(peers []flow.PeerHint) error {
	if len(peers) > 32 {
		return ErrInvalid
	}
	data, err := json.Marshal(peers)
	if err != nil {
		return err
	}
	value := C.CString(string(data))
	defer C.free(unsafe.Pointer(value))
	return codeToErr(C.lt_torrent_restore_peers(t.id, value))
}
