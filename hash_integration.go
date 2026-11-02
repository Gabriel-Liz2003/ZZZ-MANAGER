package main

import (
 "context"
 "fmt"
 "strings"
)

func (a *App) syncCommunityHashDatabase(ctx context.Context)error{
 raw,db,err:=downloadCommunityHashDatabase(ctx)
 if err!=nil{
  a.mu.Lock()
  if a.hashCatalog==nil{a.hashStatus="Falha na sincronização; use o botão Atualizar hashes"}
  a.mu.Unlock()
  return err
 }
 if err:=persistCommunityHashDatabase(a.configFile,raw);err!=nil{return fmt.Errorf("Não foi possível gravar cache de hashes: %w",err)}
 cat:=catalogFromDatabase(db)
 if cat.HashCount<50{return fmt.Errorf("Base baixada sem hashes aproveitáveis")}
 a.mu.Lock()
 a.hashCatalog=cat
 a.hashStatus=fmt.Sprintf("%d hashes de modelos sincronizados",cat.HashCount)
 a.mu.Unlock()
 return nil
}
func (a *App) extraOutfits()[]OfficialOutfit{
 if a.hashCatalog==nil{return nil}
 return a.hashCatalog.Outfits
}
func (a *App) findOfficialOutfit(agent,id string)(OfficialOutfit,bool){
 if o,ok:=lookupOfficialCostume(agent,id);ok{return o,true}
 for _,o:=range a.extraOutfits(){if strings.EqualFold(o.Agent,agent)&&o.ID==id{return o,true}}
 return OfficialOutfit{},false
}
func (a *App) hasOfficialOutfit(agent,id string)bool{
 _,ok:=a.findOfficialOutfit(agent,id);return ok
}

// Detection is computed on every library scan, including disabled mods.
// User overrides win; the classifier never edits a mod's INI or folder.
func (a *App) enrichMods(mods []*Mod){
 for _,m:=range mods{
  det:=identifyFromModelHashes(gatherModelHashes(m.path),a.hashCatalog)
  m.OutfitID="unknown"
  m.OutfitSource="unknown"
  m.OutfitConfidence=det.Confidence
  m.OutfitEvidence=det.Evidence
  m.OutfitStatus=det.Status
  explicitAgent:=a.config.Tags[m.ID]
  if det.Agent!=""{
   if explicitAgent==""{m.Character=det.Agent
   }else if !strings.EqualFold(explicitAgent,det.Agent){
    m.OutfitStatus="Conflito: agente escolhido não corresponde aos hashes"
    det.OutfitID="unknown"
   }
  }
  if manual,ok:=a.config.OutfitTags[m.ID];ok{
   if manual=="unknown" || a.hasOfficialOutfit(m.Character,manual){
    m.OutfitID=manual
    m.OutfitSource="manual"
    m.OutfitConfidence="manual"
    m.OutfitStatus="Classificado manualmente"
    continue
   }
  }
  if det.OutfitID!="unknown" && a.hasOfficialOutfit(m.Character,det.OutfitID){
   m.OutfitID=det.OutfitID
   m.OutfitSource="hash"
  }else if det.OutfitID!="unknown"{
   m.OutfitStatus="Traje identificado nos hashes, mas ausente no catálogo local"
  }
 }
}
