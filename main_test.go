package main

import (
	"archive/zip"
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeMods(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Mods")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	return root
}
func addTestMod(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(p, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, "mod.ini"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func TestScanToggleProfilesAndConflicts(t *testing.T) {
	root := makeMods(t)
	addTestMod(t, root, "Nangong Casual", "[TextureOverrideA]\nhash = deadbeef\n")
	addTestMod(t, root, "Miyabi Dress", "[TextureOverrideX]\nhash= DEADBEEF\n")
	addTestMod(t, root, "DISABLED_Rina Office", "[TextureOverrideX]\nhash= deadbeef\n")
	mods, c, err := scanMods(root, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 3 || len(c) != 1 {
		t.Fatalf("mods=%d conflicts=%v", len(mods), c)
	}
	if c[0].Hash != "deadbeef" || len(c[0].ModNames) != 2 {
		t.Fatal(c)
	}
	r := findMod(mods, "Rina Office")
	if r == nil || r.Enabled {
		t.Fatalf("expected disabled rina, %v", r)
	}
	if err := toggleMod(r, true); err != nil {
		t.Fatal(err)
	}
	mods, c, err = scanMods(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 3 || len(c) != 1 || len(c[0].ModNames) != 3 {
		t.Fatalf("after enable mods=%d conflicts=%v", len(mods), c)
	}
	m := findMod(mods, "Nangong Casual")
	if m == nil || m.Character != "Nangong" {
		t.Fatal(m)
	}
	if err := toggleMod(m, false); err != nil {
		t.Fatal(err)
	}
	mods, _, err = scanMods(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	m = findMod(mods, "Nangong Casual")
	if m == nil || m.Enabled {
		t.Fatal(m)
	}
	if _, err := os.Stat(filepath.Join(root, "DISABLED_Nangong Casual", "mod.ini")); err != nil {
		t.Fatal(err)
	}
}
func makeZIP(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, value := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestZIPImportAndNoOverwrite(t *testing.T) {
	root := makeMods(t)
	blob := makeZIP(t, map[string]string{"CoolMod/Effect.ini": "[TextureOverride]\nhash=abc12345", "CoolMod/preview.png": "hello"})
	name, err := importZIP(root, "Ellen Skin.zip", bytes.NewReader(blob))
	if err != nil {
		t.Fatal(err)
	}
	if name != "Ellen Skin" {
		t.Fatal(name)
	}
	if _, err := os.Stat(filepath.Join(root, name, "CoolMod", "Effect.ini")); err != nil {
		t.Fatal(err)
	}
	name, err = importZIP(root, "Ellen Skin.zip", bytes.NewReader(blob))
	if err != nil {
		t.Fatal(err)
	}
	if name != "Ellen Skin (2)" {
		t.Fatal(name)
	}
	mods, _, err := scanMods(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 2 {
		t.Fatalf("expected 2 mods, got %d", len(mods))
	}
}
func TestZIPTraversalAndBadFiles(t *testing.T) {
	root := makeMods(t)
	cases := []map[string]string{{"../escape.ini": "bad"}, {"folder/../../escape.ini": "bad"}, {"C:\\Windows\\bad.ini": "bad"}, {"foo.exe": "not a mod"}, {"foo/con.ini": "bad"}, {"foo/evil.ini": "good", "../evil.ini": "bad"}}
	for _, files := range cases {
		blob := makeZIP(t, files)
		if _, err := importZIP(root, "Bad.zip", bytes.NewReader(blob)); err == nil {
			t.Fatalf("expected reject %v", files)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("import left files: %v", entries)
	}
}
func TestInheritedDisabled(t *testing.T) {
	root := makeMods(t)
	addTestMod(t, root, filepath.Join("DISABLED_Character", "ModTest"), "hash=12345678")
	mods, _, err := scanMods(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatal(mods)
	}
	if !mods[0].InheritedDisabled || mods[0].Enabled {
		t.Fatal(mods[0])
	}
	err = toggleMod(mods[0], true)
	if err == nil || !strings.Contains(err.Error(), "pasta acima") {
		t.Fatal(err)
	}
}

// Regression: a library without conflicts must serialize as [] (not null),
// so the browser can read .length without crashing on the initial screen.
func TestNoConflictsReturnsArray(t *testing.T) {
	root := makeMods(t)
	addTestMod(t, root, "Nangong Outfit", "[TextureOverrideA]\nhash=deadbeef\n")
	_, conflicts, err := scanMods(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if conflicts == nil || len(conflicts) != 0 {
		t.Fatalf("expected empty non-nil conflicts, got %#v", conflicts)
	}
}

// Regression: per-agent apply must leave every other character's mods alone.
func TestApplyAgentProfileIsScoped(t *testing.T) {
	root := makeMods(t)
	addTestMod(t, root, "Nangong One", "[TextureOverrideA]\nhash=deadbeef\n")
	addTestMod(t, root, "Nangong Two", "[TextureOverrideB]\nhash=deadbeea\n")
	addTestMod(t, root, "Rina Dress", "[TextureOverrideC]\nhash=deadbeec\n")
	a := &App{token: "test", configFile: filepath.Join(t.TempDir(), "config.json"), config: Config{ModDir: root, Tags: map[string]string{}, Profiles: map[string][]string{"Nangong": {"Nangong One"}}}}
	body := strings.NewReader(`{"name":"Nangong","scope":"Nangong"}`)
	req := httptest.NewRequest("POST", "/api/apply-profile", body)
	req.Header.Set("X-Desk-Token", "test")
	rec := httptest.NewRecorder()
	a.api(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "DISABLED_Nangong Two")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "Rina Dress")); err != nil {
		t.Fatal(err)
	}
}
