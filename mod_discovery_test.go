package main

import (
 "os"
 "net/http/httptest"
 "strings"
 "path/filepath"
 "testing"
)

func writeModFile(t *testing.T, root, rel, value string) {
 t.Helper()
 path:=filepath.Join(root, filepath.FromSlash(rel))
 if err:=os.MkdirAll(filepath.Dir(path),0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(path, []byte(value),0644);err!=nil{t.Fatal(err)}
}

func TestRoxyOverallsPackageKeepsResourcesAndTogglesTogether(t *testing.T) {
 root:=makeMods(t)
 writeModFile(t,root,"RoxyOveralls/RoxyOveralls/toggles/Roxy.ini","[KeyShowMenu1]\nkey = h\n[TextureOverride]\nhash = aabbccdd\n")
 writeModFile(t,root,"RoxyOveralls/RoxyOveralls/DISABLED_resources/Roxy.ini","[KeyOldMenu]\nkey = v\n[TextureOverride]\nhash = ccddeeff\n")
 writeModFile(t,root,"RoxyOveralls/RoxyOveralls/Press H or F to open toggle menu.txt","press H")
 writeModFile(t,root,"RoxyOveralls/RoxyOveralls/textures/image.png","png")

 mods,conflicts,err:=scanMods(root,nil)
 if err!=nil{t.Fatal(err)}
 if len(mods)!=1 {t.Fatalf("expected ONE RoxyOveralls mod, got %d: %#v",len(mods),mods)}
 mod:=mods[0]
 if mod.ID!="RoxyOveralls" || mod.Name!="RoxyOveralls" || mod.Character!="Roxy" || !mod.Enabled {
  t.Fatalf("wrong root/character: %#v",mod)
 }
 if mod.IniCount!=2 {t.Fatalf("should count package INIs, got %d",mod.IniCount)}
 if len(mod.Hotkeys)!=2 || mod.Hotkeys[0].Keys[0]!="h" || mod.Hotkeys[0].Disabled || mod.Hotkeys[1].Keys[0]!="v" || !mod.Hotkeys[1].Disabled {t.Fatalf("active and disabled commands not correctly distinguished: %#v",mod.Hotkeys)}
 if mod.hashes["ccddeeff"] || !mod.hashes["aabbccdd"] {t.Fatal("disabled resources should not contribute hashes")}
 if len(conflicts)!=0 {t.Fatalf("nested resources may not produce duplicate conflicts: %#v",conflicts)}
 if err:=toggleMod(mod,false);err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,"DISABLED_RoxyOveralls","RoxyOveralls","toggles","Roxy.ini"));err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,"DISABLED_RoxyOveralls","RoxyOveralls","DISABLED_resources","Roxy.ini"));err!=nil{t.Fatal(err)}
 mods,_,err=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 if len(mods)!=1 || mods[0].Enabled {t.Fatal("disabled package should remain a single disabled mod")}
 if err:=toggleMod(mods[0],true);err!=nil{t.Fatal(err)}
 mods,_,err=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 if len(mods)!=1 || !mods[0].Enabled {t.Fatal("reenabling should restore single package")}
}

func TestNestedResourceDirectoriesWithoutDuplicateWrapper(t *testing.T) {
 root:=makeMods(t)
 writeModFile(t,root,"Roxy outfit/toggles/skin.ini","[KeyNext]\nkey = y\n")
 writeModFile(t,root,"Roxy outfit/resources/assets.ini","[TextureOverride]\nhash = cafebabe\n")
 mods,_,err:=scanMods(root,nil)
 if err!=nil {t.Fatal(err)}
 if len(mods)!=1 || mods[0].Name!="Roxy outfit" || mods[0].IniCount!=2 {t.Fatalf("failed to group resources + toggles: %#v",mods)}
}

func TestCategoryFolderKeepsIndependentMods(t *testing.T) {
 root:=makeMods(t)
 writeModFile(t,root,"Rina/Outfit A/config.ini","[TextureOverrideA]\nhash=deadbeef\n")
 writeModFile(t,root,"Rina/Outfit B/config.ini","[TextureOverrideB]\nhash=deadbeea\n")
 mods,_,err:=scanMods(root,nil)
 if err!=nil {t.Fatal(err)}
 if len(mods)!=2 {t.Fatalf("Rina category must preserve separate mods, got %d",len(mods))}
 if mods[0].ID!="Rina/Outfit A" || mods[1].ID!="Rina/Outfit B" {t.Fatalf("wrong mods: %q / %q",mods[0].ID,mods[1].ID)}
}

func TestNestedPackageConflictOnlyBetweenActualMods(t *testing.T) {
 root:=makeMods(t)
 writeModFile(t,root,"RoxyOveralls/RoxyOveralls/toggles/a.ini","[TextureOverrideA]\nhash = deadbeef\n")
 writeModFile(t,root,"RoxyOveralls/RoxyOveralls/resources/b.ini","[TextureOverrideB]\nhash = deadbeef\n")
 writeModFile(t,root,"Other Outfit/mod.ini","[TextureOverrideC]\nhash = deadbeef\n")
 mods,conflicts,err:=scanMods(root,nil)
 if err!=nil {t.Fatal(err)}
 if len(mods)!=2 || len(conflicts)!=1 {t.Fatalf("expected two real mods and one cross-mod conflict: %d / %#v",len(mods),conflicts)}
 if len(conflicts[0].ModIDs)!=2 {t.Fatal("nested resources/toggles must not count as independent conflict owners")}
}

func TestLegacyResourcesProfileRemainsApplicable(t *testing.T) {
 root:=makeMods(t)
 writeModFile(t,root,"RoxyOveralls/RoxyOveralls/toggles/Roxy.ini","[KeyMenu]\nkey = h\n")
 writeModFile(t,root,"RoxyOveralls/RoxyOveralls/resources/Roxy.ini","[TextureOverride]\nhash=12345678\n")
 writeModFile(t,root,"Rina Dress/mod.ini","[TextureOverride]\nhash=87654321\n")
 mods,_,err:=scanMods(root,nil)
 if err!=nil {t.Fatal(err)}
 if len(mods)!=2 {t.Fatalf("expected consolidated Roxy plus Rina, got %d",len(mods))}
 selected:=findModForSavedProfile(mods,"RoxyOveralls/RoxyOveralls/resources")
 if selected==nil || selected.ID!="RoxyOveralls" {t.Fatalf("legacy ID not resolved: %#v",selected)}
 // Simulate real API applying an old saved preset with a child-folder ID.
 a:=&App{token:"test",configFile:filepath.Join(t.TempDir(),"config.json"),config:Config{
  ModDir:root,Tags:map[string]string{},Profiles:map[string][]string{
    "Roxy":{"RoxyOveralls/RoxyOveralls/resources"},
  },
 }}
 req:=httptest.NewRequest("POST","/api/apply-profile",strings.NewReader(`{"name":"Roxy","scope":"Roxy"}`))
 req.Header.Set("X-Desk-Token","test")
 rec:=httptest.NewRecorder()
 a.api(rec,req)
 if rec.Code!=200 {t.Fatalf("old profile should apply: %d %s",rec.Code,rec.Body.String())}
 if _,err:=os.Stat(filepath.Join(root,"Rina Dress","mod.ini"));err!=nil{t.Fatal("Rina mod affected",err)}
 if _,err:=os.Stat(filepath.Join(root,"RoxyOveralls","RoxyOveralls","toggles","Roxy.ini"));err!=nil{t.Fatal(err)}
}

func TestDisabledNestedHCFolderShortcutIsShownButInactive(t *testing.T) {
 root:=makeMods(t)
 name:="[HC] Roxy - OG Edit & Bunny Hoodie"
 path:=name+"/"+name+"/DISABLED_"+name+"/Toggles/Roxy.ini"
 writeModFile(t,root,path,"[KeyToggleHoodie]\nkey = VK_F10\ntype = cycle\n")
 writeModFile(t,root,name+"/"+name+"/resources/Body.ini","[TextureOverrideBody]\nhash=deadbeef\n")
 mods,conflicts,err:=scanMods(root,nil)
 if err!=nil {t.Fatal(err)}
 if len(mods)!=1 {t.Fatalf("expected one package, got %d",len(mods))}
 if mods[0].Character!="Roxy" || !mods[0].Enabled {t.Fatalf("wrong character or mod status: %#v",mods[0])}
 keys:=mods[0].Hotkeys
 if len(keys)!=1 || keys[0].Keys[0]!="VK_F10" || !keys[0].Disabled {
  t.Fatalf("disabled nested shortcut must still appear, marked inactive: %#v",keys)
 }
 if len(conflicts)!=0 {t.Fatalf("false conflicts: %#v",conflicts)}
}
func TestDisabledTopLevelModStillShowsKeysAsInactive(t *testing.T) {
 root:=makeMods(t)
 writeModFile(t,root,"DISABLED_Roxy Dress/mod.ini","[KeyCycleOutfit]\nkey = VK_HOME\n")
 mods,_,err:=scanMods(root,nil)
 if err!=nil {t.Fatal(err)}
 if len(mods)!=1 || len(mods[0].Hotkeys)!=1 || !mods[0].Hotkeys[0].Disabled {
  t.Fatalf("disabled mod should show only informational keys: %#v",mods)
 }
}
