package msx

import (
	"context"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"net/http/httptest"
	"server/settings"
	"strings"
	"sync"
	"testing"
	"time"
)

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error { b.closed = true; return nil }

func TestResponseBodyClosesAfterForwarding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	body := &trackedBody{Reader: strings.NewReader("response")}
	rsp(ctx, &http.Response{StatusCode: 200, Body: body, Header: http.Header{}, ContentLength: 8}, nil)
	if !body.closed {
		t.Fatal("upstream body leaked")
	}
}

func TestProxyCancellationAndRedirectGuard(t *testing.T) {
	previous := settings.BTsets()
	config := settings.NewDefaultConfig()
	config.Flow.MSXAllowLAN = true
	settings.StoreBTsets(config)
	t.Cleanup(func() { settings.StoreBTsets(previous); lanTransport.CloseIdleConnections() })
	started := make(chan struct{})
	cancelled := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://169.254.169.254/", 302)
			return
		}
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}))
	defer upstream.Close()
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	requestCtx, cancel := context.WithCancel(context.Background())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/msx/proxy", nil).WithContext(requestCtx)
	done := make(chan error, 1)
	go func() {
		response, err := remote(ctx, http.MethodGet, upstream.URL, nil)
		if response != nil {
			response.Body.Close()
		}
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request never started")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled request succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("request ignored cancellation")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("upstream connection remained open")
	}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/msx/proxy", nil)
	response, err := remote(ctx, http.MethodGet, upstream.URL+"/redirect", nil)
	if response != nil {
		response.Body.Close()
	}
	if err == nil {
		t.Fatal("redirect bypassed destination policy")
	}
}

func TestLauncherConcurrentReadsAndWrites(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := settings.HttpAuth
	settings.HttpAuth = false
	t.Cleanup(func() { settings.HttpAuth = previous })
	router := gin.New()
	SetupRoute(router)
	var workers sync.WaitGroup
	for i := 0; i < 40; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for _, method := range []string{http.MethodPost, http.MethodGet} {
				request := httptest.NewRequest(method, "/msx/start.json", strings.NewReader(`"menu:test"`))
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				if response.Code != http.StatusOK {
					t.Errorf("launcher status %d", response.Code)
				}
			}
		}()
	}
	workers.Wait()
}
