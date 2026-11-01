package main

import (
 "encoding/binary"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "unicode/utf16"
)

func TestParseZZMIHotkeys(t *testing.T) {
 content:=`[Constants]
global $hoodie = 0
[KeyToggleHoodie]
key = ctrl shift h ; alternate clothing
type = cycle
$hoodie = 0, 1, 2

[KeyHairCycle]
key = VK_F6
key = XB_RIGHT_SHOULDER
back = VK_F5
type = cycle
condition = $active == 1
; [KeyDisabled]
; key = 8

[TextureOverride]
key = x
[KeyboardNotAnActualKey]
key = z
[KeyHold]
key = alt 9
type = hold
`
 keys:=parseKeybinds([]byte(content),"Casual/mod.ini")
 if len(keys)!=3{t.Fatalf("want 3 bindings, got %#v",keys)}
 if keys[0].Label!="Toggle Hoodie" || keys[0].Keys[0]!="ctrl shift h" || keys[0].Mode!="cycle"{t.Fatal(keys[0])}
 if len(keys[1].Keys)!=2 || keys[1].Keys[1]!="XB_RIGHT_SHOULDER" || keys[1].Back[0]!="VK_F5" || keys[1].Condition!="$active == 1"{t.Fatal(keys[1])}
 if keys[2].Keys[0]!="alt 9" || keys[2].Mode!="hold"{t.Fatal(keys[2])}
}

func TestUTF16LEHotkeys(t *testing.T) {
 src:="[KeyNextSkin]\nkey = VK_RIGHT\ntype = toggle\n"
 units:=utf16.Encode([]rune(src))
 b:=[]byte{0xff,0xfe}
 for _,u:=range units {
  pair:=make([]byte,2);binary.LittleEndian.PutUint16(pair,u);b=append(b,pair...)
 }
 result:=parseKeybinds(b,"m.ini")
 if len(result)!=1 || result[0].Keys[0]!="VK_RIGHT" {t.Fatal(result)}
}

func TestReadKeybindsWhileDisabled(t *testing.T) {
 root:=makeMods(t)
 target:=filepath.Join(root,"DISABLED_Miyabi Casual")
 if err:=os.MkdirAll(target,0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(target,"mod.ini"),[]byte("[KeySwitchOutfit]\nkey = F10\n"),0644);err!=nil{t.Fatal(err)}
 mods,_,err:=scanMods(root,nil)
 if err!=nil{t.Fatal(err)}
 if len(mods)!=1 || mods[0].Enabled || len(mods[0].Hotkeys)!=1 || mods[0].Hotkeys[0].Keys[0]!="F10" {t.Fatal(mods)}
}

func TestINIWithOnlyCommentedKeys(t *testing.T){
 all:=parseKeybinds([]byte("; [KeyNo]\n; key = x\n[KeyEmpty]\n; key = y\n[KeyReal]\nkey = v # active\n"),"mod.ini")
 if len(all)!=1 || all[0].Keys[0]!="v"{t.Fatal(all)}
}

func TestMaxINIHotkeySize(t *testing.T){
 root:=makeMods(t)
 path:=filepath.Join(root,"Huge.ini")
 if err:=os.WriteFile(path,[]byte(strings.Repeat("a",maxHotkeyINIBytes+1)),0644);err!=nil{t.Fatal(err)}
 got:=readModHotkeys(path,root,nil)
 if len(got)!=0{t.Fatal(got)}
}
