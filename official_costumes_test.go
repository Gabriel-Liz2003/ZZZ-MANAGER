package main
import("os";"path/filepath";"testing")
func TestAstraOfficialCostumesAreDifferentFromMods(t *testing.T){
 outfits:=officialCostumesFor("Astra")
 if len(outfits)!=2 || outfits[0].ID!="base" || outfits[1].ID!="chandelier" {t.Fatalf("Astra outfits: %#v",outfits)}
 if outfits[0].Name!="Traje padrão"||outfits[1].Name!="Chandelier" {t.Fatal(outfits)}
 if _,valid:=lookupOfficialCostume("Ellen","chandelier");valid {t.Fatal("cross-agent costume accepted")}
}
func TestOutfitTagKeepsModsSeparate(t *testing.T){
 root:=makeMods(t)
 addTestMod(t,root,"Astra Base Mod","[TextureOverride]\nhash=abcdef12\n")
 addTestMod(t,root,"DISABLED_Astra Chandelier Mod","[TextureOverride]\nhash=abcdef13\n")
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 for _,m:=range mods {m.OutfitID=normalizedOutfitTag(m,nil)}
 if mods[0].OutfitID!="base"||mods[1].OutfitID!="chandelier"{t.Fatalf("wrong groups: %#v",mods)}
 changes,pick,err:=planOutfitSelection(mods,"Astra","chandelier","Astra Chandelier Mod")
 if err!=nil{t.Fatal(err)}
 if !pick.MustEquipInGame||pick.Outfit!="Chandelier"{t.Fatalf("misleading selection %#v",pick)}
 if err=applyWardrobeChanges(changes);err!=nil{t.Fatal(err)}
 for _,name:=range []string{"DISABLED_Astra Base Mod","Astra Chandelier Mod"}{
  if _,err:=os.Stat(filepath.Join(root,name,"mod.ini"));err!=nil{t.Fatal(name,err)}
 }
}
func TestRandomOnlyOwnedOfficialCostumes(t *testing.T){
 root:=makeMods(t)
 addTestMod(t,root,"Astra Base Mod","[TextureOverride]\nhash=abcdef12\n")
 addTestMod(t,root,"DISABLED_Astra Chandelier Mod","[TextureOverride]\nhash=abcdef13\n")
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 for _,m:=range mods{m.OutfitID=normalizedOutfitTag(m,nil)}
 // Without owning Chandelier, base must be the only eligible costume.
 changes,picks,err:=planOfficialOutfitRandom(mods,map[string]bool{},func(n int)(int,error){return n-1,nil})
 if err!=nil||len(picks)!=1||picks[0].OutfitID!="base" {t.Fatalf("unowned costume selected: %#v %v",picks,err)}
 _=changes
 // Mark owned: Chandelier now participates.
 _,picks,err=planOfficialOutfitRandom(mods,map[string]bool{"Astra/chandelier":true},func(n int)(int,error){return n-1,nil})
 if err!=nil||picks[0].OutfitID!="chandelier"||picks[0].ModID!="Astra Chandelier Mod"{t.Fatalf("wanted owned Chandelier: %#v %v",picks,err)}
}
func TestNoModOfficialCostumeDoesNotPretendToEquipGameOutfit(t *testing.T){
 changes,pick,err:=planOutfitSelection(nil,"Astra","chandelier","")
 if err!=nil||len(changes)!=0||!pick.MustEquipInGame{t.Fatal(err,pick)}
}
