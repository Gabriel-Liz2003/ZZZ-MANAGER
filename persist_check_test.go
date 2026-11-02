package main
import("os";"path/filepath";"testing")
func TestDetectPersistVariablesWithoutModifyingINI(t *testing.T){
 root:=makeMods(t)
 source:="[Constants]\nglobal persist $glasses = 1\n; global persist $ignored=0\nglobal $hair = 1\n[KeyGlasses]\nkey = J\n$glasses = 0,1\n"
 mod:=filepath.Join(root,"Roxy Outfit")
 if err:=os.MkdirAll(mod,0755);err!=nil{t.Fatal(err)}
 path:=filepath.Join(mod,"mod.ini")
 if err:=os.WriteFile(path,[]byte(source),0644);err!=nil{t.Fatal(err)}
 if n:=countPersistInModDir(mod);n!=1{t.Fatalf("wanted one persistent global, got %d",n)}
 mods,_,err:=scanMods(root,nil)
 if err!=nil||len(mods)!=1||mods[0].PersistentVariables!=1{t.Fatalf("missing persistence stats %v %#v",err,mods)}
 after,err:=os.ReadFile(path);if err!=nil||string(after)!=source{t.Fatal("reading persistence must not edit mod")}
}
func TestPersistDoesNotCountDisabledNestedVariant(t *testing.T){
 root:=makeMods(t)
 mod:=filepath.Join(root,"Roxy Overalls")
 if err:=os.MkdirAll(filepath.Join(mod,"DISABLED_old"),0755);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(mod,"config.ini"),[]byte("[Constants]\nglobal $glasses = 1\n"),0644);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(mod,"DISABLED_old","old.ini"),[]byte("[Constants]\nglobal persist $glasses = 0\n"),0644);err!=nil{t.Fatal(err)}
 mods,_,err:=scanMods(root,nil)
 if err!=nil||len(mods)!=1||mods[0].PersistentVariables!=0{t.Fatalf("disabled nested persistence must not count %v %#v",err,mods)}
}
