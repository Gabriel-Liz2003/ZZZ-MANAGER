package main
import("os";"path/filepath";"strings";"testing")
func TestSaveInGamePersistedVariations(t *testing.T){
 root:=makeMods(t)
 addTestMod(t,root,"Roxy Outfit","[Constants]\nglobal persist $glasses = 1\nglobal $hair = 0\n[KeyToggleGlasses]\nkey=J\n")
 runtime:="$\\Mods\\Roxy Outfit\\mod.ini\\glasses = 0\n$\\Mods\\Roxy Outfit\\mod.ini\\hair = 9\n"
 p:=filepath.Join(filepath.Dir(root),"d3dx_user.ini")
 if err:=os.WriteFile(p,[]byte(runtime),0644);err!=nil{t.Fatal(err)}
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 if len(mods)!=1||mods[0].PersistentVariables!=1{t.Fatalf("missing persisted definition: %#v",mods)}
 count,err:=syncModPersistentDefaults(root,mods[0])
 if err!=nil{t.Fatal(err)}
 if count!=1{t.Fatalf("count %d",count)}
 output,err:=os.ReadFile(filepath.Join(root,"Roxy Outfit","mod.ini"));if err!=nil{t.Fatal(err)}
 if !strings.Contains(string(output),"global persist $glasses = 0")||!strings.Contains(string(output),"global $hair = 0"){t.Fatalf("unexpected modified ini: %s",output)}
 backup,err:=os.ReadFile(filepath.Join(root,"Roxy Outfit","mod.ini.zzzmanager.bak"));if err!=nil{t.Fatal(err)}
 if !strings.Contains(string(backup),"global persist $glasses = 1"){t.Fatal("backup missing original value")}
 still,err:=os.ReadFile(p);if err!=nil||string(still)!=runtime{t.Fatal("must not write live d3dx_user")}
}
func TestSaveAppearanceWithoutRuntimeValueDoesNotModifyMod(t *testing.T){
 root:=makeMods(t)
 addTestMod(t,root,"Roxy Outfit","[Constants]\nglobal persist $glasses = 1\n")
 if err:=os.WriteFile(filepath.Join(filepath.Dir(root),"d3dx_user.ini"),[]byte("$\\Mods\\Other\\mod.ini\\glasses = 0\n"),0644);err!=nil{t.Fatal(err)}
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 if _,err:=syncModPersistentDefaults(root,mods[0]);err==nil{t.Fatal("unmatched namespace should not save")}
 b,err:=os.ReadFile(filepath.Join(root,"Roxy Outfit","mod.ini"));if err!=nil||!strings.Contains(string(b),"$glasses = 1"){t.Fatal("state modified without matching key")}
 if _,err:=os.Stat(filepath.Join(root,"Roxy Outfit","mod.ini.zzzmanager.bak"));!os.IsNotExist(err){t.Fatal("backup created without valid changes")}
}
func TestSaveAppearanceIsDisallowedWhenModDisabled(t *testing.T){
 root:=makeMods(t);addTestMod(t,root,"DISABLED_Roxy Outfit","[Constants]\nglobal persist $glasses=1\n")
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 if _,err:=syncModPersistentDefaults(root,mods[0]);err==nil{t.Fatal("disabled skin should not save")}
}
