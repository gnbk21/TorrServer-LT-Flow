package utils

import (
	"context"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"time"

	_ "golang.org/x/image/webp"
)

// CheckImgUrl validates a supported image header without allocating its pixels.
// Network/temporary server failures are uncertain: preserve an existing poster.
// verified does not promise that the entire remote image can be decoded.
func CheckImgUrl(link string) (ok, verified bool) {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return false, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		return false, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return true, false
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return true, false
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, false
	}

	limitedReader := io.LimitReader(resp.Body, 2*1024*1024)

	config, _, err := image.DecodeConfig(limitedReader)
	if err != nil {
		if ctx.Err() != nil {
			return true, false
		}
		return false, false
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width) > (32<<20)/int64(config.Height) {
		return false, false
	}
	return true, true
}

// SelectPoster avoids repeated network checks for unchanged artwork. Clearing
// the field is explicit; a failed replacement retains the previous value.
func SelectPoster(candidate, previous string) string {
	if candidate == "" || candidate == previous {
		return candidate
	}
	ok, verified := CheckImgUrl(candidate)
	if !ok || (previous != "" && !verified) {
		return previous
	}
	return candidate
}
