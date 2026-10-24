package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const LogFileBytes int64 = 10 << 20
const LogArchives = 2 // active + two archives; 30 MiB per distinct sink

// Keep the open descriptor: native stdout/stderr and shared Go loggers retain
// the same sink across rotation. A small watcher also bounds native-only output.
// Copy/truncate cannot promise lossless native writes concurrent with rotation.
type rotatingLog struct {
	mu   sync.Mutex
	file *os.File
	path string
	stop chan struct{}
	done chan struct{}
}

var rotations []*rotatingLog

func regularLog(path string) error {
	info, err := os.Lstat(path)
	if err == nil && !info.Mode().IsRegular() {
		return errors.New("log destination must be a regular file")
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
func newRotatingLog(file *os.File) *rotatingLog {
	r := &rotatingLog{file: file, path: file.Name(), stop: make(chan struct{}), done: make(chan struct{})}
	rotations = append(rotations, r)
	go func() {
		defer close(r.done)
		tick := time.NewTicker(250 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-r.stop:
				return
			case <-tick.C:
				r.mu.Lock()
				_ = r.rotate(0)
				r.mu.Unlock()
			}
		}
	}()
	return r
}
func (r *rotatingLog) rotate(incoming int64) error {
	info, err := r.file.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 || info.Size()+incoming <= LogFileBytes {
		return nil
	}
	for i := 1; i <= LogArchives; i++ {
		if err = regularLog(fmt.Sprintf("%s.%d", r.path, i)); err != nil {
			return err
		}
	}
	last := fmt.Sprintf("%s.%d", r.path, LogArchives)
	if err = os.Remove(last); err != nil && !os.IsNotExist(err) {
		return err
	}
	for i := LogArchives - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", r.path, i)
		if err = os.Rename(src, fmt.Sprintf("%s.%d", r.path, i+1)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	archive, err := os.OpenFile(r.path+".1", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	// A native burst may exceed the threshold between watcher ticks. Keep only
	// the bounded tail; no directory scan or unrelated file is ever modified.
	start := max(int64(0), info.Size()-LogFileBytes)
	_, err = io.Copy(archive, io.NewSectionReader(r.file, start, info.Size()-start))
	if err == nil {
		err = archive.Sync()
	}
	err = errors.Join(err, archive.Close())
	if err != nil {
		return err
	}
	// Windows append-only handles cannot truncate. Open a separate checked
	// handle without changing the descriptor retained by native writers.
	truncate, err := os.OpenFile(r.path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer truncate.Close()
	target, err := truncate.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(info, target) {
		return errors.New("log destination changed during rotation")
	}
	return truncate.Truncate(0)
}
func (r *rotatingLog) Write(input []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	original := len(input)
	if int64(len(input)) > LogFileBytes {
		input = input[len(input)-int(LogFileBytes):]
	}
	if err := r.rotate(int64(len(input))); err != nil {
		return 0, err
	}
	n, err := r.file.Write(input)
	if err != nil {
		return 0, err
	}
	if n != len(input) {
		return 0, io.ErrShortWrite
	}
	return original, nil
}
func stopRotations() {
	for _, r := range rotations {
		close(r.stop)
		<-r.done
	}
	rotations = nil
}
