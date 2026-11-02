package main

import(
 "fmt"
 "os"
 "path/filepath"
 "testing"
)
func exampleHashDatabase() *hashDatabase{
 chars:=make([]hashCharacter,40)
 for i:=range chars {chars[i]=hashCharacter{Name:fmt.Sprintf("Unused Agent %d",i),Skins:[]hashSkin{{ID:"base",Name:"Default",Components:map[string]hashComponent{}}}}}
 chars[0]=hashCharacter{Name:"Astra Yao",Aliases:[]string{"Astra"},BaseSkinID:"scarlet_rock",Skins:[]hashSkin{
  {ID:"scarlet_rock",Name:"Scarlet Rock",Components:map[string]hashComponent{
   "Torso":{DrawVB:"11111111",PositionVB:"22222222",IB:"33333333"},
   "Hair":{DrawVB:"55555555",PositionVB:"66666666"},
  }},
  {ID:"chandelier",Name:"Chandelier",Components:map[string]hashComponent{
   "Body":{DrawVB:"645d075e",PositionVB:"43f58537",IB:"02d8a2cb"},
   "Hair":{DrawVB:"55555555",PositionVB:"66666666"},
  }},
 }}
 chars[1]=hashCharacter{Name:"Nicole Demara",Aliases:[]string{"Nicole"},BaseSkinID:"lil_sassy",Skins:[]hashSkin{
  {ID:"lil_sassy",Name:"Lil Sassy",Components:map[string]hashComponent{"Body":{DrawVB:"77777777",IB:"88888888"}}},
 }}
 return &hashDatabase{Characters:chars}
}
func TestHashDetectionDistinguishesAstraOutfits(t *testing.T){
 cat:=catalogFromDatabase(exampleHashDatabase())
 cases:=[]struct{hashes []string;want,confidence string}{
  {[]string{"11111111","22222222"},"base","high"},
  {[]string{"645d075e","43f58537"},"chandelier","high"},
  {[]string{"645d075e"},"chandelier","medium"},
  {[]string{"55555555","66666666"},"unknown","none"},
  {[]string{"11111111","645d075e"},"unknown","none"},
  {[]string{"deadbeef"},"unknown","none"},
 }
 for _,c:=range cases{
  set:=map[string]bool{}
  for _,h:=range c.hashes{set[h]=true}
  got:=identifyFromModelHashes(set,cat)
  if got.OutfitID!=c.want||got.Confidence!=c.confidence{t.Errorf("hashes=%v got=%#v want outfit=%s confidence=%s",c.hashes,got,c.want,c.confidence)}
 }
}
func TestHashesIgnoreINICommentsAndDisabledResources(t *testing.T){
 root:=makeMods(t)
 p:=filepath.Join(root,"DISABLED_Astra Chandelier Test")
 if err:=os.MkdirAll(filepath.Join(p,"DISABLED_old"),0755);err!=nil{t.Fatal(err)}
 data:="[TextureOverrideBody]\nhash = 645d075e\nhash = 43f58537 ; comment\n; hash = deadbeef\n[KeySomething]\nkey = H\n"
 if err:=os.WriteFile(filepath.Join(p,"mod.ini"),[]byte(data),0644);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(p,"DISABLED_old","old.ini"),[]byte("hash = 11111111\n"),0644);err!=nil{t.Fatal(err)}
 observed:=gatherModelHashes(p)
 if !observed["645d075e"]||!observed["43f58537"]||observed["11111111"]||observed["deadbeef"]{t.Fatalf("incorrect parsed hashes %#v",observed)}
 mods,_,err:=scanMods(root,nil);if err!=nil||len(mods)!=1{t.Fatalf("scan: %v %#v",err,mods)}
 a:=&App{hashCatalog:catalogFromDatabase(exampleHashDatabase()),config:Config{Tags:map[string]string{},OutfitTags:map[string]string{}}}
 a.enrichMods(mods)
 if mods[0].OutfitID!="chandelier"||mods[0].OutfitSource!="hash"||mods[0].OutfitEvidence!=2{t.Fatalf("failed detection %#v",mods[0])}
}
func TestManualOutfitOverrideAndUnknown(t *testing.T){
 root:=makeMods(t)
 addTestMod(t,root,"Astra Costume","[TextureOverride]\nhash = 645d075e\n")
 mods,_,err:=scanMods(root,nil);if err!=nil{t.Fatal(err)}
 a:=&App{hashCatalog:catalogFromDatabase(exampleHashDatabase()),config:Config{Tags:map[string]string{},OutfitTags:map[string]string{"Astra Costume":"base"}}}
 a.enrichMods(mods)
 if mods[0].OutfitID!="base"||mods[0].OutfitSource!="manual"{t.Fatal("manual classification lost",mods[0])}
 delete(a.config.OutfitTags,"Astra Costume")
 a.enrichMods(mods)
 if mods[0].OutfitID!="chandelier"||mods[0].OutfitSource!="hash"{t.Fatal("hash classification missing",mods[0])}
 a.hashCatalog=nil
 a.enrichMods(mods)
 if mods[0].OutfitID!="unknown"{t.Fatal("must not assume base when database unavailable")}
}
func TestCommunitySkinMappingToExistingOutfitIDs(t *testing.T){
 db:=exampleHashDatabase()
 cat:=catalogFromDatabase(db)
 outfit:=OfficialOutfit{}
 found:=false
 for _,o:=range cat.Outfits{if o.Agent=="Astra"&&o.Name=="Chandelier"{outfit=o;found=true}}
 if !found||outfit.ID!="chandelier"{t.Fatalf("wrong catalog IDs: %#v",cat.Outfits)}
 if cat.HashCount<5{t.Fatalf("not enough indexed hashes %d",cat.HashCount)}
}
