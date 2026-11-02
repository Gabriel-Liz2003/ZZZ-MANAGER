package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestEditModHotkeyPreservesOtherFieldsAndCreatesBackup(t *testing.T) {
	root := makeMods(t)
	content := "[Constants]\r\nvalue = 1\r\n[KeySwap0]\r\nkey = VK_NUMPAD3 ; original\r\ntype = cycle\r\ncondition = $active0 == 1\r\n[KeySwap1]\r\nkey = l\r\n"
	dir := filepath.Join(root, "Nangong Costume")
	if err := os.MkdirAll(dir, 0755); err != nil { t.Fatal(err) }
	path := filepath.Join(dir, "nangong.ini")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil { t.Fatal(err) }
	mods, _, err := scanMods(root, nil)
	if err != nil || len(mods) != 1 { t.Fatalf("%v: %v", err, mods) }
	h := mods[0].Hotkeys[0]
	if err := editModHotkey(mods[0], hotkeyChange{File:h.File, Section:h.Section, Occurrence:h.Occurrence, Field:"key", Index:0, Previous:h.Keys[0], Key:"l"}); err != nil { t.Fatal(err) }
	after, err := os.ReadFile(path)
	if err != nil { t.Fatal(err) }
	expected := strings.Replace(content, "key = VK_NUMPAD3 ; original", "key = l ; original", 1)
	if string(after) != expected { t.Fatalf("unexpected ini:\n%s", after) }
	backup, err := os.ReadFile(path+".zzzmanager.bak")
	if err != nil || string(backup) != content { t.Fatalf("original backup missing: %v", err) }
	if err := editModHotkey(mods[0], hotkeyChange{File:h.File, Section:h.Section, Occurrence:h.Occurrence, Field:"key", Index:0, Previous:h.Keys[0], Key:"p"}); err == nil { t.Fatal("stale update should fail") }
}

func TestEditModHotkeyRepeatedSectionAndKey(t *testing.T) {
	root := makeMods(t)
	dir := filepath.Join(root, "Remielle")
	_ = os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "skin.ini")
	content := "[KeySwap]\nkey = f1\nkey = f2\nback = f3\n[KeySwap]\nkey = f4\n"
	_ = os.WriteFile(path, []byte(content), 0644)
	mods, _, err := scanMods(root, nil)
	if err != nil { t.Fatal(err) }
	hotkeys := mods[0].Hotkeys
	if len(hotkeys) != 2 || hotkeys[1].Occurrence != 1 { t.Fatalf("occurrences: %+v", hotkeys) }
	if err := editModHotkey(mods[0], hotkeyChange{File:hotkeys[0].File, Section:"KeySwap", Field:"back", Index:0, Previous:"f3", Key:"VK_LEFT"}); err != nil { t.Fatal(err) }
	mods, _, _ = scanMods(root, nil)
	if err := editModHotkey(mods[0], hotkeyChange{File:hotkeys[1].File, Section:"KeySwap", Occurrence:1, Field:"key", Index:0, Previous:"f4", Key:"ctrl shift l"}); err != nil { t.Fatal(err) }
	got, _ := os.ReadFile(path)
	if string(got) != "[KeySwap]\nkey = f1\nkey = f2\nback = VK_LEFT\n[KeySwap]\nkey = ctrl shift l\n" { t.Fatal(string(got)) }
}

func TestEditModHotkeyUTF16AndUnsafeInputs(t *testing.T) {
	root := makeMods(t)
	dir := filepath.Join(root, "Astra")
	_ = os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "a.ini")
	text := "[KeySwap]\r\nkey = VK_NUMPAD3\r\n"
	units := utf16.Encode([]rune(text))
	source := []byte{0xff,0xfe}
	for _, u := range units {
		pair := make([]byte,2)
		binary.LittleEndian.PutUint16(pair,u)
		source = append(source,pair...)
	}
	_ = os.WriteFile(path,source,0644)
	mods,_,err := scanMods(root,nil)
	if err != nil { t.Fatal(err) }
	h := mods[0].Hotkeys[0]
	req := hotkeyChange{File:h.File,Section:h.Section,Field:"key",Index:0,Previous:h.Keys[0],Key:"l"}
	if err := editModHotkey(mods[0],req);err!=nil { t.Fatal(err) }
	got, _ := os.ReadFile(path)
	if !bytes.HasPrefix(got,[]byte{0xff,0xfe}) { t.Fatal("UTF16 BOM lost") }
	result := parseKeybinds(got,"a.ini")
	if len(result)!=1 || result[0].Keys[0]!="l" { t.Fatal(result) }
	req.Key="l\n[Constants]\nfoo = 1"
	if err := editModHotkey(mods[0],req);err==nil { t.Fatal("accepted ini injection") }
	req.Key="p";req.File="../outside.ini"
	if err := editModHotkey(mods[0],req);err==nil { t.Fatal("accepted arbitrary path") }
}

func TestValidNewHotkey(t *testing.T) {
	for _, v := range []string{"l","8","F12","VK_NUMPAD3","ctrl shift l","VK_LEFT","alt VK_SPACE"} {
		if !validNewHotkey(v) { t.Errorf("rejected %q",v) }
	}
	for _, v := range []string{"","ctrl","ctrl ctrl l","x;foo","l\nkey = k","f25","VK_NOTREAL","shift meta l"} {
		if validNewHotkey(v) { t.Errorf("accepted %q",v) }
	}
}
