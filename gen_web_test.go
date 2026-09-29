package main

import "testing"

func TestAssetCacheControl(t *testing.T) {
	for path, want := range map[string]string{"/index.html": "no-cache", "/site.webmanifest": "no-cache", "/sw.js": "no-cache", "/assets/index-B5-HiPwS.js": "public, max-age=31536000, immutable", "/icon.png": "public, max-age=3600", "/lord-icon-2.0.2.js": "public, max-age=3600"} {
		if got := assetCacheControl(path); got != want {
			t.Errorf("%s: got %s, want %s", path, got, want)
		}
	}
}
