package flow

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Whole boundary pieces count against disk quota, even when a file begins or
// ends inside them. Duplicate/overlapping selections reserve their union once.
type PieceReservation struct {
	Hash                   string
	First, Last            int
	PieceLength, TotalSize int64
}

func PreparationReserved(ranges []PieceReservation) (int64, error) {
	byHash := make(map[string][]PieceReservation)
	for _, r := range ranges {
		if len(r.Hash) != 40 || r.First < 0 || r.Last < r.First || r.PieceLength <= 0 || r.TotalSize <= 0 || int64(r.Last) > (r.TotalSize-1)/r.PieceLength {
			return 0, errors.New("invalid preparation geometry")
		}
		byHash[r.Hash] = append(byHash[r.Hash], r)
	}
	var total int64
	for _, rs := range byHash {
		// At most 16 jobs; interval boundaries avoid allocating a torrent bitmap.
		for i, r := range rs {
			segments := [][2]int{{r.First, r.Last}}
			for _, old := range rs[:i] {
				if old.PieceLength != r.PieceLength || old.TotalSize != r.TotalSize {
					return 0, errors.New("inconsistent preparation geometry")
				}
				var next [][2]int
				for _, s := range segments {
					if old.Last < s[0] || old.First > s[1] {
						next = append(next, s)
						continue
					}
					if s[0] < old.First {
						next = append(next, [2]int{s[0], old.First - 1})
					}
					if s[1] > old.Last {
						next = append(next, [2]int{old.Last + 1, s[1]})
					}
				}
				segments = next
			}
			for _, s := range segments {
				n := min(r.TotalSize, int64(s[1]+1)*r.PieceLength) - int64(s[0])*r.PieceLength
				if n < 0 || total > 1<<60-n {
					return 0, errors.New("preparation quota overflow")
				}
				total += n
			}
		}
	}
	return total, nil
}

const MaxPreparationStateBytes = 128 << 20

// State includes private metadata and authorized tracker passkeys; only its
// dedicated DTO is returned by the API. Atomic replacement survives interruption.
func WritePreparationState(name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > MaxPreparationStateBytes {
		return errors.New("preparation state too large")
	}
	if err = os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(name), ".flow-preparation-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temp, name)
}

func ReadPreparationState(name string, value any) error {
	st, err := os.Lstat(name)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Size() > MaxPreparationStateBytes {
		return errors.New("invalid preparation state file")
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
