package main
import (
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func TestWardrobeSelectSingleSkinPerAgent(t *testing.T){
 root:=makeMods(t)
 addTestMod(t,root,"Roxy Dress A","[TextureOverride]\nhash=deadbeef")
 addTestMod(t,root,"DISABLED_Roxy Dress B","[TextureOverride]\nhash=deadbeea")
 addTestMod(t,root,"Rina Casual","[TextureOverride]\nhash=deadbeeb")
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 changes,choice,err:=selectWardrobe(mods,"Roxy","Roxy Dress B")
 if err!=nil{t.Fatal(err)}
 if choice.Name!="Roxy Dress B"||len(changes)!=2{t.Fatalf("invalid selection %#v %#v",choice,changes)}
 if err:=applyWardrobeChanges(changes);err!=nil{t.Fatal(err)}
 for _,name:=range []string{"DISABLED_Roxy Dress A","Roxy Dress B","Rina Casual"} {
  if _,err:=os.Stat(filepath.Join(root,name,"mod.ini"));err!=nil{t.Fatalf("missing folder %s: %v",name,err)}
 }
 mods,_,err=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 changes,choice,err=selectWardrobe(mods,"Roxy","")
 if err!=nil||choice.ModID!=""{t.Fatal(err,choice)}
 if err:=applyWardrobeChanges(changes);err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,"DISABLED_Roxy Dress B","mod.ini"));err!=nil{t.Fatal(err)}
}
func TestRandomWardrobeOnlyTouchesAgentsWithActiveSkins(t *testing.T){
 root:=makeMods(t)
 addTestMod(t,root,"Roxy A","hash=abcdef12")
 addTestMod(t,root,"DISABLED_Roxy B","hash=abcdef11")
 addTestMod(t,root,"Rina A","hash=abcdef13")
 addTestMod(t,root,"DISABLED_Ellen A","hash=abcdef14")
 addTestMod(t,root,"Some Uncategorized Mod","hash=abcdef15")
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 changes,choices,err:=planRandomWardrobe(mods,func(n int)(int,error){return n-1,nil})
 if err!=nil{t.Fatal(err)}
 if len(choices)!=2{t.Fatalf("Expected Rina and Roxy only: %#v",choices)}
 if err:=applyWardrobeChanges(changes);err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,"Roxy B","mod.ini"));err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,"DISABLED_Roxy A","mod.ini"));err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,"DISABLED_Rina A","mod.ini"));err!=nil{t.Fatal(err)}
 for _,name:=range []string{"DISABLED_Ellen A","Some Uncategorized Mod"}{
  if _,err:=os.Stat(filepath.Join(root,name,"mod.ini"));err!=nil{t.Fatalf("changed noneligible %s",name)}
 }
}
func TestRandomWardrobeIncludesOriginal(t *testing.T){
 root:=makeMods(t);addTestMod(t,root,"Roxy A","hash=12345678")
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 changes,choices,err:=planRandomWardrobe(mods,func(n int)(int,error){if n!=1{t.Fatalf("expected original-only choice, got %d",n)};return 0,nil})
 if err!=nil||len(choices)!=1||choices[0].ModID!=""{t.Fatal(err,choices)}
 if err:=applyWardrobeChanges(changes);err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,"DISABLED_Roxy A","mod.ini"));err!=nil{t.Fatal(err)}
}
func TestWardrobeRejectsUnknownOrCrossAgentSelection(t *testing.T){
 root:=makeMods(t);addTestMod(t,root,"Roxy Outfit","hash=12345678");addTestMod(t,root,"Miyabi Outfit","hash=12345679")
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 for _,agent:=range []string{"Sem categoria","", "FakeAgent"}{if _,_,err=selectWardrobe(mods,agent,"");err==nil{t.Fatal("accepted agent",agent)}}
 if _,_,err=selectWardrobe(mods,"Roxy","Miyabi Outfit");err==nil{t.Fatal("cross-agent outfit accepted")}
}
func TestWardrobeRollsBackIfLaterFolderRenameFails(t *testing.T){
 root:=makeMods(t);addTestMod(t,root,"Roxy A","hash=12345678");addTestMod(t,root,"DISABLED_Roxy B","hash=12345679")
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 changes,_,err:=selectWardrobe(mods,"Roxy","Roxy B")
 if err!=nil{t.Fatal(err)}
 // Block B's destination so the second rename fails after A is disabled.
 if err:=os.Mkdir(filepath.Join(root,"Roxy B"),0755);err!=nil{t.Fatal(err)}
 if err:=applyWardrobeChanges(changes);err==nil||!strings.Contains(err.Error(),"revert"){t.Fatal("expected rollback",err)}
 if _,err:=os.Stat(filepath.Join(root,"Roxy A","mod.ini"));err!=nil{t.Fatal("rollback failed",err)}
}
