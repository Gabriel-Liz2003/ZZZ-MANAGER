package main

import (
 "errors"
 "fmt"
 "sort"
 "strings"
)

// OfficialOutfit is an IN-GAME costume, not a third-party mod.
// Selecting it in this app configures mod compatibility only: the player
// must equip the purchased/unlocked official costume inside Zenless Zone Zero.
type OfficialOutfit struct {
 Agent string `json:"agent"`
 ID string `json:"id"`
 Name string `json:"name"`
 Image string `json:"image"`
 Base bool `json:"base"`
}
var specialOfficialOutfits=[]OfficialOutfit{
 {Agent:"Astra",ID:"chandelier",Name:"Chandelier",Image:"data:image/webp;base64,UklGRp4FAABXRUJQVlA4IJIFAADQHgCdASpVAHsAPxF6rlCsKCQirrrtiYAiCUAZ+pXh4XuPf7YVNizqZeOmzacP30laHwGxV0NL9AjGFhzTfhAMIXfEWeEaNeiDeVuVpMSjgkCdy449WZmdSPQP/PMR0XGXB7ro530CQ4fMZ+0RxTLdhgvb5/n1qiSPFfFFECMOZsJ15PDJ2qiXKPjWOFVfa9gmUvv+zvaRGTHIZve1N4loLTR5SwavJMdWaspjHs/ok0txfFrU/wDe8DGG3N+V7uWUmd73+y7MVe+F3TRQsS0vkd/TDo3zvE6x31vXBAK4ZtDsC1TNzbKwfbU2lwNhtUJp7VCZGDEd/FMAmH088IAA/vfkYU6kUOmPVK8gSrgoasgtfVUoD8GcJmAYAKBGukCvRdOJa6UCXlswhrE+ZBg/AgO/bAmKXBxRBlkkQDFgdxp3gyCZGbGDHsJMn5mzAAmLv2qEYQ0cr8bxtPwBnDCYaBStqqMbbOV7tj8TBnNIx32Kkmz7T51DLkNSht9mBu6ywgko7K0es+xQW3E4tB55l8FnRDWAgwLX8HNLqxh0hFNgKxXFDTC8UhAfJZleV9yr5fU5cRO24i8+hA9Sp6a+BdWG93jiJfgufJRtXZfTbU5blAgm4R2NDrTGYuM1c4LQoXm1xei/IaQbL2qHR0DxuYnTMNr6M3FQYCHhnAgSNGCmu3sA9W4h18ziWMoIwOhKXGVlLz2PDZvv5Imd3Kbd+T2I0YpWWchLhPvEAVXYHJ9bPpLVOzImnbP4NhxjPaugq4E4VjXm/YMyCMMxrS4/sF3OYlL5LpQB4f3twk5pWYx6e9AVi8fxemzveWLeJiUzejxCBI91P7Wmo5DaZrz7a0nQrvIL03pGcSl1SWHUZcj3z0Dx4bzU0bfpBZBs044NM9ydR0LcJAsLoJF+BcSmvVjP6I3XR0+WcTPg+2YihLIcYSN2dCO8EkqCaHY5iXpVHQeaId8kO/51woAFhcX8E5aNHeFh6smqNKBx0Tb41wG9aKhFkgpewCUWRa2yPn3GUNxA5FVeDN/ElhBA0KfkFy9Nogam7gDMXE7JsMruuLJf5b9d3aACEAV+1au+b8Z4ozkCOsTi5qr5H952W3nyWK4Q0nRgp323+qnnbjpZaVRCSSG6pFfPZ+St17IS9R6/cy60XIyD+eef3weLDtT5EqIMdjuLU3D9Bht8EZDh2DA912+XWLvYLh8qEggJ2Y3wUm/L7k7ekPz2MgxcSJtbyzHQudoG54XRu0QXBr67DzyRevaZ8IQn2XZZg6cfBCRn6sqhKyD6cDDUfnv50TpiXczM7tD3X2Q7C+Dqy+/znwG0HvPy2rvRwfBc+pTcI423faceEILtINKINXUruDHlIz4MGEbRtf2MM9S2JANxQHQSjB2e3RBaow9uTSrFYcnoMOrVr1ZwMenKG98/mmNzmFfTh20EAIWCxhohPkdwtUhpLApqTRWqi9s0vKtHWhIvsJF2JrLAkZv1Pza08V1nOp3QtyZ7KZmuCcxSwyKCN1xYvJwjVDTe8lXdkD3A++jbxYjADKG03wulUFIZJiY0UhQAXQhB+N5kAlZFTB00D+92dWHrpaGr/1+tQaSnb7Ii2Sz2Xo5NBCwR6Wykrg0t2Lcx2BuqHLU7Uu8axJakZepcRErYnKTm1/j3GWKxheolW7p0DNUuQdkGkJAk1JUQmSxh9hO03inxjepu6ZUBgMUD35XWOtrQUh19vC7rKkuD4IKkf5RGncG2d5s1z0Osk5j1kNsrNMFy6h2a0mRl3W0yVcflQtUxhLdVw9Nj4miD8zA7iiZ6gqePCJsQDpid7XcAgKHdcyuRMoSCRLRcmYfXnZpCbuYLa0IOiV1aLCDk8aFiyQDOuETjjY5vavdyWpthbCLXd5WDSZ78ZnAgAAAA"},
 {Agent:"Ellen",ID:"on-campus",Name:"On Campus",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Ellen%20Joe%20On%20Campus%20Portrait.png"},
 {Agent:"Nicole",ID:"cunning-cutie",Name:"Cunning Cutie",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Nicole%20Demara%20Cunning%20Cutie%20Portrait.png"},
 {Agent:"Jane",ID:"nocturne-of-light",Name:"Nocturne of Light",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Jane%20Doe%20Nocturne%20of%20Light%20Portrait.png"},
 {Agent:"Vivian",ID:"iris-of-the-shore",Name:"Iris of the Shore",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Vivian%20Banshee%20Iris%20of%20the%20Shore%20Portrait.png"},
 {Agent:"Yixuan",ID:"trails-of-ink",Name:"Trails of Ink",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Yixuan%20Trails%20of%20Ink%20Portrait.png"},
 {Agent:"Alice",ID:"sea-of-thyme",Name:"Sea of Thyme",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Alice%20Thymefield%20Sea%20of%20Thyme%20Portrait.png"},
 {Agent:"Yuzuha",ID:"tanuki-in-broad-daylight",Name:"Tanuki in Broad Daylight",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Ukinami%20Yuzuha%20Tanuki%20in%20Broad%20Daylight%20Portrait.png"},
 {Agent:"Miyabi",ID:"dignified-blossom",Name:"Dignified Blossom",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Hoshimi%20Miyabi%20Dignified%20Blossom%20Portrait.png"},
 {Agent:"Nangong",ID:"rhapsodys-muse",Name:"Rhapsody's Muse",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Nangong%20Yu%20Rhapsody's%20Muse%20Portrait.png"},
 {Agent:"Aria",ID:"discordant-note",Name:"Discordant Note",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Aria%20Discordant%20Note%20Portrait.png"},
 {Agent:"Sunna",ID:"afternoon-tea-break",Name:"Afternoon Tea Break",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Sunna%20Afternoon%20Tea%20Break%20Portrait.png"},
 {Agent:"Ye Shunguang",ID:"touch-of-dawnlight",Name:"Touch of Dawnlight",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Ye%20Shunguang%20Touch%20of%20Dawnlight%20Portrait.png"},
 {Agent:"Velina",ID:"shade-of-leisure",Name:"Shade of Leisure",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Velina%20Airgid%20Shade%20of%20Leisure%20Portrait.png"},
 {Agent:"Pan Yinhu",ID:"culinary-jewel",Name:"Culinary Jewel",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Pan%20Yinhu%20Culinary%20Jewel%20Portrait.png"},
 {Agent:"Komano Manato",ID:"white-heart-silhouette",Name:"White Heart Silhouette",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Komano%20Manato%20White%20Heart%20Silhouette%20Portrait.png"},
 {Agent:"Belle",ID:"delicate-sunlight",Name:"Delicate Sunlight",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Belle%20Delicate%20Sunlight%20Portrait.png"},
 {Agent:"Belle",ID:"summer-skies",Name:"Summer Skies",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Belle%20Summer%20Skies%20Portrait.png"},
 {Agent:"Belle",ID:"brilliance-of-stars",Name:"Brilliance of Stars",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Belle%20Brilliance%20of%20Stars%20Portrait.png"},
 {Agent:"Wise",ID:"soaring-crane",Name:"Soaring Crane",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Wise%20Soaring%20Crane%20Portrait.png"},
 {Agent:"Wise",ID:"peaceful-waves",Name:"Peaceful Waves",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Wise%20Peaceful%20Waves%20Portrait.png"},
 {Agent:"Wise",ID:"oath-of-skies",Name:"Oath of Skies",Image:"https://zenless-zone-zero.fandom.com/wiki/Special:FilePath/Agent%20Wise%20Oath%20of%20Skies%20Portrait.png"},
 {Agent:"Phoenix",ID:"daydream-complex",Name:"Daydream Complex",Image:""},
 {Agent:"Phoenix",ID:"the-illness-called-life",Name:"The Illness Called Life",Image:""},
 {Agent:"Severian",ID:"solemnly-impeccable",Name:"Solemnly Impeccable",Image:""},
}
func officialCostumesFor(agent string)[]OfficialOutfit {
 result:=[]OfficialOutfit{{Agent:agent,ID:"base",Name:"Traje padrão",Base:true}}
 for _,o:=range specialOfficialOutfits{if strings.EqualFold(o.Agent,agent){result=append(result,o)}}
 return result
}
func lookupOfficialCostume(agent,id string)(OfficialOutfit,bool){
 for _,o:=range officialCostumesFor(agent){if o.ID==id{return o,true}}
 return OfficialOutfit{},false
}
func defaultOutfitTag(m *Mod) string{
 lower:=strings.ToLower(strings.NewReplacer("_"," ","-"," ","."," ").Replace(m.Name+" "+m.RelPath))
 for _,o:=range specialOfficialOutfits{
  if !strings.EqualFold(o.Agent,m.Character){continue}
  needle:=strings.ToLower(strings.NewReplacer("-"," ","'","").Replace(o.Name))
  if strings.Contains(lower,needle)||strings.Contains(lower,strings.ReplaceAll(o.ID,"-"," ")){return o.ID}
 }
 return "base" // unclassified mods require user confirmation
}
func normalizedOutfitTag(m *Mod,tags map[string]string)string{
 if tag,ok:=tags[m.ID];ok {
  if _,valid:=lookupOfficialCostume(m.Character,tag);valid{return tag}
 }
 return defaultOutfitTag(m)
}
type OutfitPick struct{
 Agent string `json:"agent"`
 OutfitID string `json:"outfitId"`
 Outfit string `json:"outfit"`
 ModID string `json:"modId"`
 ModName string `json:"modName"`
 MustEquipInGame bool `json:"mustEquipInGame"`
}
func planOutfitSelection(mods []*Mod,agent,outfitID,modID string)([]wardrobeChange,OutfitPick,error){
 if !knownWardrobeAgent(agent){return nil,OutfitPick{},errors.New("Agente inválido")}
 costume,valid:=lookupOfficialCostume(agent,outfitID)
 if !valid{return nil,OutfitPick{},errors.New("Traje oficial não encontrado para este agente")}
 group:=wardrobeGroup(mods,agent)
 // The agent may have no installed mods; the selection still updates
 // its metadata and tells the user to equip the costume in game.
 pick:=OutfitPick{Agent:agent,OutfitID:outfitID,Outfit:costume.Name,MustEquipInGame:!costume.Base}
 if modID!=""{
  found:=false
  for _,m:=range group {
   if m.ID!=modID{continue}
   if m.OutfitID!=outfitID{return nil,OutfitPick{},errors.New("Esse mod pertence a outro traje oficial")}
   if m.InheritedDisabled{return nil,OutfitPick{},errors.New("Mod dentro de pasta superior desativada")}
   pick.ModID=m.ID;pick.ModName=m.Name;found=true;break
  }
  if !found{return nil,OutfitPick{},errors.New("Mod não pertence ao agente informado")}
 }
 changes:=[]wardrobeChange{}
 for _,m:=range group{if m.Enabled&&m.ID!=modID{changes=append(changes,wardrobeChange{mod:m,enabled:false})}}
 for _,m:=range group{if !m.Enabled&&m.ID==modID{changes=append(changes,wardrobeChange{mod:m,enabled:true})}}
 return changes,pick,nil
}
// Only include the base outfit and official outfits explicitly marked owned.
// Randomization chooses ONE outfit, and at most one of that outfit's mods.
// Characters with no currently enabled mods are ignored.
func planOfficialOutfitRandom(mods []*Mod,owned map[string]bool,choose func(int)(int,error))([]wardrobeChange,[]OutfitPick,error){
 agents:=map[string]bool{}
 for _,m:=range mods{if m.Enabled&&knownWardrobeAgent(m.Character){agents[m.Character]=true}}
 names:=[]string{};for a:=range agents{names=append(names,a)};sort.Strings(names)
 if len(names)==0{return nil,nil,errors.New("Nenhum agente com mod ativo para sortear")}
 changes:=[]wardrobeChange{};picks:=[]OutfitPick{}
 for _,agent:=range names{
  // Only outfits with at least one installed compatible mod participate.
  // This preserves exactly one enabled mod per eligible agent.
  candidates:=[]struct{outfit OfficialOutfit; variants []string}{}
  for _,o:=range officialCostumesFor(agent){
   if !o.Base&&!owned[agent+"/"+o.ID]{continue}
   variants:=[]string{}
   for _,m:=range wardrobeGroup(mods,agent){if m.OutfitID==o.ID&&!m.InheritedDisabled{variants=append(variants,m.ID)}}
   if len(variants)>0{candidates=append(candidates,struct{outfit OfficialOutfit;variants []string}{o,variants})}
  }
  if len(candidates)==0{return nil,nil,fmt.Errorf("%s não possui mods compatíveis com trajes liberados para o sorteio",agent)}
  n,err:=choose(len(candidates));if err!=nil{return nil,nil,err}
  if n<0||n>=len(candidates){return nil,nil,errors.New("Resultado inválido do sorteio")}
  outfit:=candidates[n].outfit
  variants:=candidates[n].variants
  j,err:=choose(len(variants));if err!=nil{return nil,nil,err}
  if j<0||j>=len(variants){return nil,nil,errors.New("Variante sorteada inválida")}
  step,pick,err:=planOutfitSelection(mods,agent,outfit.ID,variants[j])
  if err!=nil{return nil,nil,fmt.Errorf("%s: %w",agent,err)}
  changes=append(changes,step...);picks=append(picks,pick)
 }
 return changes,picks,nil
}
