package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTransferReaderBoundsAndReusedBuffer(t *testing.T) {
	r := newFR(7)
	b := make([]byte, 5)
	for _, want := range []int{5, 2, 0} {
		start := r.pos
		for i := range b {
			b[i] = 0xff
		}
		n, err := r.Read(b)
		if n != want || (err == io.EOF) != (want == 0) {
			t.Fatalf("read %d: %d %v", want, n, err)
		}
		for i := 0; i < n; i++ {
			if b[i] != byte((start+int64(i))*31) {
				t.Fatal("stale buffer bytes")
			}
		}
	}
	if _, err := r.Seek(-1, io.SeekStart); err == nil {
		t.Fatal("negative seek accepted")
	}
	if _, err := r.Seek(0, 3); err == nil {
		t.Fatal("unknown seek origin accepted")
	}
	if _, err := r.Seek(-2, io.SeekEnd); err != nil {
		t.Fatal(err)
	}
	if n, _ := r.Read(b); n != 2 {
		t.Fatal("seek did not preserve bounds")
	}
}

func TestLANTransferHTTPBoundsAndBusy(t *testing.T) {
	router := gin.New()
	router.GET("/flow/lan-test", flowLANTest)
	server := httptest.NewServer(router)
	defer server.Close()
	for _, value := range []string{"0", "33", "-1", "bad", "99999999999999999999999"} {
		response, err := http.Get(server.URL + "/flow/lan-test?mib=" + value)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 400 {
			t.Fatalf("%s accepted: %d", value, response.StatusCode)
		}
	}
	response, err := http.Get(server.URL + "/flow/lan-test?mib=1")
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || len(data) != 1<<20 {
		t.Fatalf("transfer: %d %d %v", response.StatusCode, len(data), err)
	}
	if response.Header.Get("Cache-Control") != "no-store, no-transform" {
		t.Fatal("transfer can be cached")
	}
	for i, b := range data {
		if b != byte(i*31) {
			t.Fatalf("payload changed at %d", i)
		}
	}
	lanTestSlots <- struct{}{}
	lanTestSlots <- struct{}{}
	defer func() { <-lanTestSlots; <-lanTestSlots }()
	response, err = http.Get(server.URL + "/flow/lan-test?mib=1")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 429 {
		t.Fatal("concurrency limit ignored")
	}
}

func TestLANTransferRequiresDeadline(t *testing.T) {
	router := gin.New()
	router.GET("/flow/lan-test", flowLANTest)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/flow/lan-test?mib=1", nil))
	if response.Code != 503 {
		t.Fatal("unbounded writer accepted")
	}
}

func TestExistingSpeedDownloadPreservesLargeRangeSupport(t *testing.T) {
	router := gin.New()
	router.GET("/download/:size", download)
	request := httptest.NewRequest("GET", "/download/2048", nil)
	request.Header.Set("Range", "bytes=2147483647-2147483647")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 206 || response.Body.Len() != 1 || response.Body.Bytes()[0] != byte(255*31%256) {
		t.Fatal("existing large synthetic Range behavior changed")
	}
	for _, size := range []string{"-1", "0", "8796093022208", "bad"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", "/download/"+size, nil))
		if response.Code != 400 {
			t.Fatalf("invalid size %s accepted", size)
		}
	}
}
