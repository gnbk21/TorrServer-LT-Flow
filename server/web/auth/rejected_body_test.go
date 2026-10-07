package auth

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUnauthorizedSplitBodyConnectionClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer discardRejectedBody(w, r)
		w.Header().Set("WWW-Authenticate", "Basic realm=Authorization Required")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	address := strings.TrimPrefix(server.URL, "http://")
	for i := 0; i < 500; i++ {
		connection, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		connection.SetDeadline(time.Now().Add(3 * time.Second))
		_, err = io.WriteString(connection, "POST /settings HTTP/1.1\r\nHost: fixture\r\nConnection: close\r\nContent-Length: 16\r\n\r\n")
		if err == nil {
			_, err = io.WriteString(connection, `{"action":"get"}`)
		}
		if err != nil {
			connection.Close()
			t.Fatal(err)
		}
		response, err := http.ReadResponse(bufio.NewReader(connection), nil)
		if err != nil {
			connection.Close()
			t.Fatalf("request %d: %v", i, err)
		}
		if response.StatusCode != 401 || response.Header.Get("WWW-Authenticate") == "" {
			t.Errorf("unexpected response: %v", response.Status)
		}
		response.Body.Close()
		connection.Close()
	}
}

type rejectedWriter struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (w *rejectedWriter) SetReadDeadline(deadline time.Time) error {
	w.deadlines = append(w.deadlines, deadline)
	return nil
}

func TestRejectedBodyLimits(t *testing.T) {
	for _, test := range []struct {
		name             string
		size             int64
		close            bool
		expect, transfer string
		read             bool
	}{
		{"small", 16, true, "", "", true},
		{"keepalive", 16, false, "", "", false},
		{"unknown", -1, true, "", "", false},
		{"large", 4097, true, "", "", false},
		{"expect", 16, true, "100-continue", "", false},
		{"chunked", 16, true, "", "chunked", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := strings.NewReader(`{"action":"get"}`)
			request := httptest.NewRequest("POST", "/settings", body)
			request.ContentLength, request.Close = test.size, test.close
			request.Header.Set("Expect", test.expect)
			if test.transfer != "" {
				request.TransferEncoding = []string{test.transfer}
			}
			writer := &rejectedWriter{ResponseRecorder: httptest.NewRecorder()}
			discardRejectedBody(writer, request)
			if (body.Len() == 0) != test.read {
				t.Fatalf("consumed body=%v, want %v", body.Len() == 0, test.read)
			}
			if test.read && (len(writer.deadlines) != 2 || !writer.deadlines[1].IsZero()) {
				t.Fatal("deadline was not restored")
			}
		})
	}
	request := httptest.NewRequest("POST", "/settings", strings.NewReader("body"))
	request.Close = true
	discardRejectedBody(httptest.NewRecorder(), request)
	data, _ := io.ReadAll(request.Body)
	if string(data) != "body" {
		t.Fatal("read body without deadline support")
	}
}

func TestRejectedBodySlowSenderIsBounded(t *testing.T) {
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		discardRejectedBody(w, r)
		close(done)
	}))
	defer server.Close()
	connection, err := net.DialTimeout("tcp", strings.TrimPrefix(server.URL, "http://"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	fmt.Fprint(connection, "POST / HTTP/1.1\r\nHost: fixture\r\nConnection: close\r\nContent-Length: 16\r\n\r\n")
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("unauthorized slow sender held the handler")
	}
}
