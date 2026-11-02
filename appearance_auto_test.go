package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

func TestAppearanceWatcherRequiresNewStableRuntimeWrite(t *testing.T) {
	var w appearanceWatcher
	t0 := time.Now().Add(-20 * time.Second)
	base := runtimeFileSignature{ready: true, modTime: t0, size: 300}
	changed := runtimeFileSignature{ready: true, modTime: t0.Add(10 * time.Second), size: 300}
	if w.next("C:/Mods", base) { t.Fatal("must never replay old runtime data on launch") }
	if w.next("C:/Mods", base) { t.Fatal("no modification yet") }
	if w.next("C:/Mods", changed) { t.Fatal("first observation is not stable") }
	if !w.next("C:/Mods", changed) { t.Fatal("two consecutive observations should trigger an auto-save") }
	if w.next("C:/Mods", changed) { t.Fatal("must not save repeatedly without further changes") }
	if w.next("D:/Mods", changed) { t.Fatal("changing library must baseline its runtime file") }
	if w.next("D:/Mods", runtimeFileSignature{}) { t.Fatal("missing file must not trigger a save") }
}

func TestRuntimePersistentValuesUTF16AndNamespaced(t *testing.T) {
	root := makeMods(t)
	content := "[Persistent]\r\n$\\Mods\\Remielle Suit\\skin.ini\\glasses = 2 ; F10\r\n" +
		"$\\Mods\\Other Suit\\skin.ini\\glasses = 1\r\n" +
		"$\\Mods\\Remielle Suit\\skin.ini\\invalid = run shell\r\n" +
		"$\\Mods\\Remielle Suit\\skin.ini\\nan = NaN\r\n"
	data := []byte{0xff, 0xfe}
	for _, v := range utf16.Encode([]rune(content)) {
		pair := make([]byte, 2)
		binary.LittleEndian.PutUint16(pair, v)
		data = append(data, pair...)
	}
	if err := os.WriteFile(runtimeINIPath(root), data, 0644); err != nil { t.Fatal(err) }
	result, err := runtimePersistentValues(root)
	if err != nil { t.Fatal(err) }
	if len(result) != 2 || result["remielle suit/skin.ini/glasses"] != "2" || result["other suit/skin.ini/glasses"] != "1" {
		t.Fatalf("bad runtime parsing: %#v", result)
	}
}

func TestAutomaticAppearanceSyncOnlyWritesMatchingActiveMod(t *testing.T) {
	root := makeMods(t)
	writeModFile(t,root,"Remielle/Remielle Outfit/skin.ini",
		"[Constants]\r\nglobal persist $glasses = 0 ; preserve\r\nglobal $unrelated = 9\r\n[KeySwap]\r\nkey = VK_NUMPAD3\r\n")
	writeModFile(t,root,"Other/Other Costume/skin.ini",
		"[Constants]\nglobal persist $glasses = 0\n")
	writeModFile(t,root,"DISABLED_Roxy Outfit/skin.ini",
		"[Constants]\nglobal persist $glasses = 0\n")
	values := map[string]string{
		"remielle/remielle outfit/skin.ini/glasses": "3",
		"other/other costume/skin.ini/glasses": "2",
		"roxy outfit/skin.ini/glasses": "9",
	}
	mods, _, err := scanMods(root,nil)
	if err != nil { t.Fatal(err) }
	for _, m := range mods {
		if !m.Enabled { continue }
		_, _, err := syncModPersistentDefaultsWithValues(root,m,values)
		if err != nil { t.Fatalf("sync %s: %v",m.ID,err) }
	}
	mod, err := os.ReadFile(filepath.Join(root,"Remielle","Remielle Outfit","skin.ini"))
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(mod),"global persist $glasses = 3 ; preserve") ||
		!strings.Contains(string(mod),"global $unrelated = 9") ||
		!strings.Contains(string(mod),"key = VK_NUMPAD3") {
		t.Fatalf("unexpected changes in mod .ini: %s",mod)
	}
	inactive, _ := os.ReadFile(filepath.Join(root,"DISABLED_Roxy Outfit","skin.ini"))
	if strings.Contains(string(inactive),"$glasses = 9") { t.Fatal("modified disabled mod") }
	backup, err := os.ReadFile(filepath.Join(root,"Remielle","Remielle Outfit","skin.ini.zzzmanager.bak"))
	if err != nil || !strings.Contains(string(backup),"$glasses = 0") { t.Fatal("original backup missing",err) }
	// Updating twice must keep the original first backup.
	values["remielle/remielle outfit/skin.ini/glasses"] = "4"
	mods, _, _ = scanMods(root,nil)
	for _, m := range mods {
		if m.Character=="Remielle" {
			_, _, err = syncModPersistentDefaultsWithValues(root,m,values)
			if err != nil { t.Fatal(err) }
		}
	}
	b, _ := os.ReadFile(filepath.Join(root,"Remielle","Remielle Outfit","skin.ini.zzzmanager.bak"))
	if string(b) != string(backup) { t.Fatal("backup overwritten with already edited data") }
}

func TestAutoSavePreservesUTF16INI(t *testing.T) {
	root := makeMods(t)
	path := filepath.Join(root, "Alice", "Outfit", "a.ini")
	if err:=os.MkdirAll(filepath.Dir(path),0755);err!=nil{t.Fatal(err)}
	source := "[Constants]\r\nglobal persist $hair = 0\r\n[KeyCycle]\r\nkey = k\r\n"
	data:=[]byte{0xff,0xfe}
	for _,v:=range utf16.Encode([]rune(source)){
		data=append(data,byte(v),byte(v>>8))
	}
	if err:=os.WriteFile(path,data,0644);err!=nil{t.Fatal(err)}
	mods,_,err:=scanMods(root,nil)
	if err!=nil||len(mods)!=1{t.Fatalf("bad mod detection %v %d",err,len(mods))}
	n,changed,err:=syncModPersistentDefaultsWithValues(root,mods[0],
		map[string]string{"alice/outfit/a.ini/hair":"1"})
	if err!=nil||n!=1||changed!=1{t.Fatalf("utf16 saved %d %d %v",n,changed,err)}
	out,err:=os.ReadFile(path)
	if err!=nil||len(out)<2||out[0]!=0xff||out[1]!=0xfe{t.Fatalf("UTF16 BOM lost: %v",err)}
	txt,_,err:=decodedHotkeyINI(out)
	if err!=nil||!strings.Contains(txt,"$hair = 1")||!strings.Contains(txt,"key = k"){
		t.Fatalf("UTF16 content damaged: %v %q",err,txt)
	}
}
