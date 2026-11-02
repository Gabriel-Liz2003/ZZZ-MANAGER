package main

import (
 "bytes"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func setupAgentOrganizer(t *testing.T,root string) *App{
 t.Helper()
 return &App{configFile:filepath.Join(t.TempDir(),"config.json"),config:Config{
  ModDir:root,Tags:map[string]string{},OutfitTags:map[string]string{},
  Profiles:map[string][]string{},SelectedOutfits:map[string]string{},OwnedOutfits:map[string]bool{},
 }}
}

func TestOrganizeModsByAgentMovesWholePackagesPreservesStatesAndProfiles(t *testing.T){
 root:=makeMods(t)
 writeModFile(t,root,"Remielle Day Outfit/body/body.ini","[TextureOverride]\nhash = aaaaaaaa\n")
 writeModFile(t,root,"Remielle Day Outfit/face/face.ini","[TextureOverride]\nhash = bbbbbbbb\n")
 writeModFile(t,root,"Remielle Day Outfit/weapon/weapon.ini","[TextureOverride]\nhash = cccccccc\n")
 writeModFile(t,root,"DISABLED_Remielle Night/mod.ini","[KeyMode]\nkey=J\n")
 writeModFile(t,root,"Anby Casual/mod.ini","[TextureOverride]\nhash = dddddddd\n")
 writeModFile(t,root,"Unknown One/mod.ini","[TextureOverride]\nhash = 12345678\n")
 a:=setupAgentOrganizer(t,root)
 a.config.Tags["Remielle Day Outfit/body"]="Remielle"
 a.config.OutfitTags["Remielle Day Outfit/body"]="base"
 a.config.Profiles["Remielle"]=[]string{"Remielle Day Outfit/body"}
 a.config.Profiles["All"]=[]string{"Remielle Day Outfit","Remielle Day Outfit/body","Anby Casual"}
 if err:=a.save();err!=nil{t.Fatal(err)}
 plan,err:=a.planAgentOrganization("")
 if err!=nil{t.Fatal(err)}
 if plan.Summary.Moved!=3||plan.Summary.Unknown!=1{t.Fatalf("Unexpected plan: %#v",plan.Summary)}
 result,err:=a.applyAgentOrganization(plan)
 if err!=nil{t.Fatal(err)}
 if result.Moved!=3{t.Fatalf("Unexpected result: %#v",result)}
 for _,p:=range []string{
  "Remielle/Remielle Day Outfit/body/body.ini",
  "Remielle/Remielle Day Outfit/face/face.ini",
  "Remielle/Remielle Day Outfit/weapon/weapon.ini",
  "Remielle/DISABLED_Remielle Night/mod.ini",
  "Anby/Anby Casual/mod.ini",
  "Unknown One/mod.ini",
 }{
  if _,err:=os.Stat(filepath.Join(root,filepath.FromSlash(p)));err!=nil{t.Fatalf("Expected file %s: %v",p,err)}
 }
 mods,_,err:=scanMods(root,a.config.Tags)
 if err!=nil||len(mods)!=4{t.Fatalf("Unexpected library after reorganization: %v %#v",err,mods)}
 m:=findMod(mods,"Remielle/Remielle Day Outfit")
 if m==nil||m.Character!="Remielle"||m.IniCount!=3||!m.Enabled{t.Fatalf("Incomplete merged package %#v",m)}
 disabled:=findMod(mods,"Remielle/Remielle Night")
 if disabled==nil||disabled.Enabled||disabled.InheritedDisabled{t.Fatalf("Lost disabled state: %#v",disabled)}
 if a.config.OutfitTags["Remielle/Remielle Day Outfit/body"]!="base"{t.Fatal("Outfit classification not migrated")}
 if a.config.Tags["Remielle/Remielle Day Outfit"]!="Remielle"{t.Fatal("New package tag missing")}
 if len(a.config.Profiles["All"])!=2||a.config.Profiles["All"][0]!="Remielle/Remielle Day Outfit"||a.config.Profiles["All"][1]!="Anby/Anby Casual"{
  t.Fatalf("Old profiles not migrated and de-duplicated: %#v",a.config.Profiles)
 }
 if len(a.config.Profiles["Remielle"])!=1||a.config.Profiles["Remielle"][0]!="Remielle/Remielle Day Outfit"{
  t.Fatalf("Legacy child preset lost: %#v",a.config.Profiles)
 }
 again,err:=a.organizeAgentFolders("")
 if err!=nil||again.Moved!=0||again.AlreadyOrganized!=3{t.Fatalf("Not idempotent: %v %#v",err,again)}
 backups,_:=filepath.Glob(a.configFile+".before-agent-folders-*.bak")
 if len(backups)==0{t.Fatal("Config backup missing")}
}

func TestOrganizerSkipsUnknownAndDisabledAncestors(t *testing.T){
 root:=makeMods(t)
 writeModFile(t,root,"DISABLED_Collection/Remielle Night/mod.ini","hash=abcdef12")
 writeModFile(t,root,"Unknown Outfit/mod.ini","hash=abcdef13")
 a:=setupAgentOrganizer(t,root)
 a.config.Tags["DISABLED_Collection/Remielle Night"]="Remielle"
 preview,err:=a.planAgentOrganization("")
 if err!=nil{t.Fatal(err)}
 if preview.Summary.Moved!=0||preview.Summary.Unknown!=1||len(preview.Summary.Skipped)!=1{
  t.Fatalf("Expected no moves: %#v",preview.Summary)
 }
 result,err:=a.applyAgentOrganization(preview)
 if err!=nil||result.Moved!=0{t.Fatal(err,result)}
 if _,err:=os.Stat(filepath.Join(root,"DISABLED_Collection","Remielle Night","mod.ini"));err!=nil{t.Fatal(err)}
}
func TestOrganizerAvoidsDuplicateNamesAndOtherMods(t *testing.T){
 root:=makeMods(t)
 writeModFile(t,root,"Remielle/Remielle Casual/mod.ini","hash=abcdef11")
 writeModFile(t,root,"Remielle Collection/Remielle Casual/mod.ini","hash=abcdef12")
 writeModFile(t,root,"DISABLED_Remielle Casual/mod.ini","hash=abcdef13")
 a:=setupAgentOrganizer(t,root)
 a.config.Tags["Remielle Collection/Remielle Casual"]="Remielle"
 result,err:=a.organizeAgentFolders("")
 if err!=nil{t.Fatal(err)}
 if result.Moved!=2{t.Fatalf("Expected 2 moved: %#v",result)}
 for _,p:=range []string{
  "Remielle/Remielle Casual/mod.ini",
  "Remielle/Remielle Casual (2)/mod.ini",
  "Remielle/DISABLED_Remielle Casual/mod.ini",
 }{
  if _,err:=os.Stat(filepath.Join(root,filepath.FromSlash(p)));err!=nil{t.Fatalf("Collision broke %s: %v",p,err)}
 }
}
func TestOrganizerKeepsNestedCollectionOfIndependentMods(t *testing.T){
 root:=makeMods(t)
 writeModFile(t,root,"Remielle costumes/Day One/mod.ini","hash=12345678")
 writeModFile(t,root,"Remielle costumes/Night Two/mod.ini","hash=87654321")
 a:=setupAgentOrganizer(t,root)
 a.config.Tags["Remielle costumes/Day One"]="Remielle"
 a.config.Tags["Remielle costumes/Night Two"]="Remielle"
 result,err:=a.organizeAgentFolders("")
 if err!=nil||result.Moved!=2{t.Fatalf("Expected independent relocations %v %#v",err,result)}
 mods,_,err:=scanMods(root,a.config.Tags)
 if err!=nil||len(mods)!=2{t.Fatalf("Distinct costumes were merged: %v %#v",err,mods)}
 for _,m:=range mods{if !strings.HasPrefix(m.ID,"Remielle/"){t.Fatalf("Mod not relocated: %#v",m)}}
}
func TestOrganizerLeavesAgentNamedModRootUntouched(t *testing.T){
 root:=makeMods(t)
 writeModFile(t,root,"Remielle/body/mod.ini","hash = abcdef12")
 writeModFile(t,root,"Remielle/face/mod.ini","hash = abcdef13")
 writeModFile(t,root,"Remielle/weapon/mod.ini","hash = abcdef14")
 writeModFile(t,root,"Remielle Extra/mod.ini","hash = abcdef15")
 a:=setupAgentOrganizer(t,root)
 result,err:=a.organizeAgentFolders("")
 if err!=nil{t.Fatal(err)}
 if result.Moved!=0||len(result.Skipped)==0{
  t.Fatalf("Cannot move an extra mod into already used mod root: %#v",result)
 }
 if _,err:=os.Stat(filepath.Join(root,"Remielle Extra","mod.ini"));err!=nil{t.Fatal(err)}
}

func TestNewZipImportsIntoRecognizedAgentFolder(t *testing.T){
 root:=makeMods(t)
 a:=setupAgentOrganizer(t,root)
 a.token="test"
 archive:=makeZIP(t,map[string]string{
  "body/body.ini":"[TextureOverride]\nhash = aaaaaaaa\n",
  "face/face.ini":"[TextureOverride]\nhash = bbbbbbbb\n",
  "weapon/weapon.ini":"[TextureOverride]\nhash = cccccccc\n",
 })
 req:=httptest.NewRequest("POST","/api/import?name=Remielle%20Dress.zip",bytes.NewReader(archive))
 req.Header.Set("X-Desk-Token","test")
 rec:=httptest.NewRecorder()
 a.api(rec,req)
 if rec.Code!=200{t.Fatalf("Import failed: %d %s",rec.Code,rec.Body.String())}
 for _,part:=range []string{"body/body.ini","face/face.ini","weapon/weapon.ini"}{
  file:=filepath.Join(root,"Remielle","Remielle Dress",filepath.FromSlash(part))
  if _,err:=os.Stat(file);err!=nil{t.Fatalf("Expected imported package component %s: %v",file,err)}
 }
 if _,err:=os.Stat(filepath.Join(root,"Remielle Dress"));!os.IsNotExist(err){t.Fatal("Empty old wrapper should be removed")}
 mods,_,err:=scanMods(root,a.config.Tags)
 if err!=nil||len(mods)!=1||mods[0].Character!="Remielle"||mods[0].IniCount!=3{
  t.Fatalf("Incorrect imported mod: %v %#v",err,mods)
 }
}

func TestChangingCharacterMovesOnlySelectedMod(t *testing.T){
 root:=makeMods(t)
 writeModFile(t,root,"Unsorted Outfit/mod.ini","hash=12345678")
 writeModFile(t,root,"Roxy Dress/mod.ini","hash=87654321")
 a:=setupAgentOrganizer(t,root)
 a.token="test"
 req:=httptest.NewRequest("POST","/api/tag",strings.NewReader(`{"id":"Unsorted Outfit","character":"Remielle"}`))
 req.Header.Set("X-Desk-Token","test")
 rec:=httptest.NewRecorder();a.api(rec,req)
 if rec.Code!=200{t.Fatalf("%d %s",rec.Code,rec.Body.String())}
 if _,err:=os.Stat(filepath.Join(root,"Remielle","Unsorted Outfit","mod.ini"));err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,"Roxy Dress","mod.ini"));err!=nil{t.Fatal("Other mods must not be moved when changing one tag",err)}
}
