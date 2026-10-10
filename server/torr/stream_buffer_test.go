package torr

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestStreamBufferLazyHeadAndCancellation(t *testing.T) {
	r := newBufferedStreamReader(bytes.NewReader(make([]byte, 2<<20)), streamBufferSize)
	w := httptest.NewRecorder()
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, httptest.NewRequest("HEAD", "http://local/video", nil), "video", time.Unix(1, 0), r)
	if r.buffer != nil {
		t.Fatal("HEAD allocated a transport buffer")
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.ctx = ctx
	if _, err := r.Read(make([]byte, 17)); err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := r.Read(make([]byte, 17)); err != context.Canceled {
		t.Fatal("buffered data ignored cancellation", err)
	}
}

// Compare the production wrapper against memory and file sources. This measures
// transport overhead and source read calls, not public-swarm throughput.
func BenchmarkStreamBuffer(b *testing.B) {
	data := bytes.Repeat([]byte("0123456789abcdef"), 1<<20)
	path := filepath.Join(b.TempDir(), "media.bin")
	if err := os.WriteFile(path, data, 0600); err != nil {
		b.Fatal(err)
	}
	for _, disk := range []bool{false, true} {
		for _, bufferSize := range []int{0, 64 << 10, 256 << 10, 1024 << 10} {
			name := "RAM/plain"
			if disk {
				name = "file/plain"
			}
			if bufferSize > 0 {
				name += "/" + strconv.Itoa(bufferSize>>10) + "KiB"
			}
			b.Run(name, func(b *testing.B) {
				var source io.ReadSeeker = bytes.NewReader(data)
				if disk {
					f, err := os.Open(path)
					if err != nil {
						b.Fatal(err)
					}
					defer f.Close()
					source = f
				}
				reads := 0
				buf := make([]byte, 32<<10)
				b.SetBytes(int64(len(data)))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := source.Seek(0, io.SeekStart); err != nil {
						b.Fatal(err)
					}
					counting := &streamCountingReader{source: source}
					var reader io.Reader = counting
					if bufferSize > 0 {
						reader = newBufferedStreamReader(counting, bufferSize)
					}
					for {
						_, err := reader.Read(buf)
						if err == io.EOF {
							break
						}
						if err != nil {
							b.Fatal(err)
						}
					}
					reads += counting.reads
				}
				b.ReportMetric(float64(reads)/float64(b.N), "source-reads/op")
			})
		}
	}
}

func TestStreamBufferReadAndSeek(t *testing.T) {
	data := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 1000)
	r := newBufferedStreamReader(bytes.NewReader(data), 4096)
	p := make([]byte, 17)
	if _, e := io.ReadFull(r, p); e != nil || !bytes.Equal(p, data[:17]) {
		t.Fatal(e)
	}
	if pos, e := r.Seek(0, io.SeekCurrent); e != nil || pos != 17 {
		t.Fatalf("current %d %v", pos, e)
	}
	if _, e := io.ReadFull(r, p); e != nil || !bytes.Equal(p, data[17:34]) {
		t.Fatal(e)
	}
	if pos, e := r.Seek(-10, io.SeekCurrent); e != nil || pos != 24 {
		t.Fatalf("relative %d %v", pos, e)
	}
	if _, e := io.ReadFull(r, p); e != nil || !bytes.Equal(p, data[24:41]) {
		t.Fatal(e)
	}
	if _, e := r.Seek(-19, io.SeekEnd); e != nil {
		t.Fatal(e)
	}
	tail, e := io.ReadAll(r)
	if e != nil || !bytes.Equal(tail, data[len(data)-19:]) {
		t.Fatal("tail", e)
	}
}

func TestStreamBufferFailedSeekKeepsUnreadData(t *testing.T) {
	data := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 100)
	r := newBufferedStreamReader(bytes.NewReader(data), 4096)
	p := make([]byte, 17)
	if _, err := io.ReadFull(r, p); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Seek(-1, io.SeekStart); err == nil {
		t.Fatal("expected failed seek")
	}
	if _, err := io.ReadFull(r, p); err != nil || !bytes.Equal(p, data[17:34]) {
		t.Fatalf("failed seek discarded unread data: %v", err)
	}
}

type streamCountingReader struct {
	source io.ReadSeeker
	reads  int
}

func (r *streamCountingReader) Read(p []byte) (int, error) {
	r.reads++
	return r.source.Read(p)
}

func (r *streamCountingReader) Seek(offset int64, whence int) (int64, error) {
	return r.source.Seek(offset, whence)
}

func TestStreamBufferAmortizesHTTPReads(t *testing.T) {
	data := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 160000)
	serve := func(buffered bool) int {
		source := &streamCountingReader{source: bytes.NewReader(data)}
		var reader io.ReadSeeker = source
		if buffered {
			reader = newBufferedStreamReader(source, 1<<20)
		}
		response := httptest.NewRecorder()
		response.Header().Set("Content-Type", "application/octet-stream")
		http.ServeContent(response, httptest.NewRequest(http.MethodGet, "http://local/video.mkv", nil), "video.mkv", time.Unix(1, 0), reader)
		if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), data) {
			t.Fatal("HTTP body changed")
		}
		return source.reads
	}
	unbuffered, buffered := serve(false), serve(true)
	if buffered*8 >= unbuffered {
		t.Fatalf("small HTTP reads were not amortized: unbuffered=%d buffered=%d", unbuffered, buffered)
	}
}

func TestStreamBufferHTTPRanges(t *testing.T) {
	data := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 1000)
	for _, rangeHeader := range []string{"bytes=13-512", "bytes=-19", "bytes=25000-", "bytes=27000-", "bytes=100-20", ""} {
		serve := func(buffered bool) *httptest.ResponseRecorder {
			req := httptest.NewRequest(http.MethodGet, "http://local/video.mkv", nil)
			req.Header.Set("Range", rangeHeader)
			resp := httptest.NewRecorder()
			var r io.ReadSeeker = bytes.NewReader(data)
			if buffered {
				r = newBufferedStreamReader(r, 4096)
			}
			http.ServeContent(resp, req, "video.mkv", time.Unix(1, 0), r)
			return resp
		}
		a, b := serve(false), serve(true)
		if a.Code != b.Code || !bytes.Equal(a.Body.Bytes(), b.Body.Bytes()) || a.Header().Get("Content-Range") != b.Header().Get("Content-Range") {
			t.Fatal(rangeHeader)
		}
	}
}

func TestStreamBufferMultipartAndHead(t *testing.T) {
	data := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 100000)
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		req := httptest.NewRequest(method, "http://local/video.mkv", nil)
		req.Header.Set("Range", "bytes=17-8191,1048589-1058589")
		a, b := httptest.NewRecorder(), httptest.NewRecorder()
		http.ServeContent(a, req, "video.mkv", time.Unix(1, 0), bytes.NewReader(data))
		http.ServeContent(b, req, "video.mkv", time.Unix(1, 0), newBufferedStreamReader(bytes.NewReader(data), 1<<20))
		if a.Code != b.Code || a.Header().Get("Content-Length") != b.Header().Get("Content-Length") {
			t.Fatal("headers differ")
		}
		if method == http.MethodHead {
			if a.Body.Len() != 0 || b.Body.Len() != 0 {
				t.Fatal("HEAD returned a body")
			}
			continue
		}
		decode := func(resp *httptest.ResponseRecorder) [][]byte {
			_, params, err := mime.ParseMediaType(resp.Header().Get("Content-Type"))
			if err != nil {
				t.Fatal(err)
			}
			reader := multipart.NewReader(bytes.NewReader(resp.Body.Bytes()), params["boundary"])
			var parts [][]byte
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(part)
				if err != nil {
					t.Fatal(err)
				}
				parts = append(parts, body)
			}
			return parts
		}
		if !reflect.DeepEqual(decode(a), decode(b)) {
			t.Fatal("multipart bytes differ")
		}
	}
}
