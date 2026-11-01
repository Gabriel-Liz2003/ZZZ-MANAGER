package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVersionOrdering(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{{"3.1.0", "v3.1.0", 0}, {"3.1.0", "v3.1.1", -1}, {"3.1.4", "3.1.2", 1}, {"3.9.0", "4.0.0", -1}} {
		got, err := compareVersion(tc.a, tc.b)
		if err != nil || got != tc.want {
			t.Fatalf("%v: got %v, %v", tc, got, err)
		}
	}
	if _, err := compareVersion("3.1", "3.1.0"); err == nil {
		t.Fatal("expected invalid version")
	}
}
func TestGithubReleaseMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "missing") {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"tag_name":"v3.2.0","html_url":"https://github.com/example","assets":[{"name":"ZZZ-Mod-Desk.exe","browser_download_url":"https://example.com/file.exe","size":123}]}`)
	}))
	defer srv.Close()
	rel, err := getLatestRelease(context.Background(), srv.Client(), srv.URL+"/latest")
	if err != nil || rel.Tag != "v3.2.0" {
		t.Fatalf("invalid release: %+v %v", rel, err)
	}
	if _, err := getLatestRelease(context.Background(), srv.Client(), srv.URL+"/missing"); err == nil {
		t.Fatal("expected 404 error")
	}
}
func TestExpectedHash(t *testing.T) {
	b := []byte("hello")
	h := sha256.Sum256(b)
	expect := hex.EncodeToString(h[:])
	got, err := expectedHash([]byte(expect+"  ZZZ-Mod-Desk.exe\n"), updateAssetName)
	if err != nil || got != expect {
		t.Fatalf("hash %q %v", got, err)
	}
	if _, err := expectedHash([]byte("wrong  ZZZ-Mod-Desk.exe"), updateAssetName); err == nil {
		t.Fatal("accepted invalid hash")
	}
}
