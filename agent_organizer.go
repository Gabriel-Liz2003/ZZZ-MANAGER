package main

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "sort"
 "strings"
 "time"
)

// AgentOrganizer moves whole packages, never individual body/face/weapon INIs.
// Existing libraries are reorganized only after explicit user confirmation.
// Newly imported or reclassified packages can be organized independently.
type OrganizeResult struct {
 Moved int `json:"moved"`
 AlreadyOrganized int `json:"alreadyOrganized"`
 Unknown int `json:"unknown"`
 Skipped []string `json:"skipped"`
 Created []string `json:"created"`
}
type agentMove struct {
 From string
 To string
 OldID string
 NewID string
 Agent string
}
type agentOrganizePlan struct{
 Changes []agentMove
 Summary OrganizeResult
}

func cleanAgentFolder(agent string) bool {
 if !knownWardrobeAgent(agent) || !zipSegmentOK(agent) {return false}
 return agent==prettyName(agent) && filepath.Base(agent)==agent
}

func (a *App) planAgentOrganization(onlyInside string)(agentOrganizePlan,error){
 plan:=agentOrganizePlan{Summary:OrganizeResult{Skipped:[]string{},Created:[]string{}}}
 mods,_,err:=scanMods(a.config.ModDir,a.config.Tags)
 if err!=nil{return plan,err}
 a.enrichMods(mods)
 root,err:=validDir(a.config.ModDir)
 if err!=nil{return plan,err}
 filter:=""
 if onlyInside!=""{
  filter,err=filepath.Abs(onlyInside);if err!=nil{return plan,err}
  rel,relErr:=filepath.Rel(root,filter)
  if relErr!=nil||rel==".."||strings.HasPrefix(rel,".."+string(filepath.Separator)){
   return plan,errors.New("Pacote importado está fora da pasta Mods")
  }
 }
 reserved:=map[string]bool{}
 for _,m:=range mods{
  if filter!=""{
   rel,err:=filepath.Rel(filter,m.path)
   if err!=nil||rel==".."||strings.HasPrefix(rel,".."+string(filepath.Separator)){continue}
  }
  agent:=strings.TrimSpace(m.Character)
  if !cleanAgentFolder(agent) {
   plan.Summary.Unknown++
   continue
  }
  // A disabled ancestor would become enabled if its child were moved out.
  if m.InheritedDisabled {
   plan.Summary.Skipped=append(plan.Summary.Skipped,m.ID+": pasta superior desativada")
   continue
  }
  rel,err:=filepath.Rel(root,m.path)
  if err!=nil{return plan,err}
  top:=strings.Split(filepath.ToSlash(rel),"/")[0]
  if strings.EqualFold(top,agent){
   plan.Summary.AlreadyOrganized++
   continue
  }
  // Never create a subdirectory inside the package being moved (e.g.
  // a mod whose entire folder happens to be named Remielle).
  targetDir:=filepath.Join(root,agent)
  inside,err:=filepath.Rel(m.path,targetDir)
  if err!=nil{return plan,err}
  if inside=="."||(!strings.HasPrefix(inside,".."+string(filepath.Separator))&&inside!=".."){
   plan.Summary.Skipped=append(plan.Summary.Skipped,m.ID+": pasta principal já ocupa o nome do agente")
   continue
  }
  if st,err:=os.Lstat(targetDir);err==nil{
   if !st.IsDir()||st.Mode()&os.ModeSymlink!=0{
    plan.Summary.Skipped=append(plan.Summary.Skipped,m.ID+": destino não é uma pasta normal")
    continue
   }
   entries,readErr:=os.ReadDir(targetDir)
   if readErr!=nil {plan.Summary.Skipped=append(plan.Summary.Skipped,m.ID+": destino inacessível");continue}
   hasOwnINI:=false
   for _,e:=range entries{
    if !e.IsDir()&&strings.EqualFold(filepath.Ext(e.Name()),".ini"){hasOwnINI=true;break}
   }
   if hasOwnINI{
    plan.Summary.Skipped=append(plan.Summary.Skipped,m.ID+": a pasta "+agent+" já contém um mod próprio")
    continue
   }
  }else if !os.IsNotExist(err){
   plan.Summary.Skipped=append(plan.Summary.Skipped,m.ID+": não foi possível verificar o destino")
   continue
  }
  base:=filepath.Base(m.path)
  dest:=""
  for i:=1;i<10001;i++{
   candidate:=base
   if i>1{candidate=fmt.Sprintf("%s (%d)",base,i)}
   if !zipSegmentOK(candidate){continue}
   p:=filepath.Join(targetDir,candidate)
   key:=strings.ToLower(p)
   if reserved[key]{continue}
   if _,err:=os.Lstat(p);os.IsNotExist(err){
    reserved[key]=true;dest=p;break
   }else if err!=nil{
    return plan,fmt.Errorf("Falha ao verificar pasta de destino: %w",err)
   }
  }
  if dest==""{
   plan.Summary.Skipped=append(plan.Summary.Skipped,m.ID+": nenhum nome livre no destino")
   continue
  }
  newID:=canonRel(filepath.ToSlash(filepath.Join(agent,filepath.Base(dest))))
  plan.Changes=append(plan.Changes,agentMove{From:m.path,To:dest,OldID:m.ID,NewID:newID,Agent:agent})
 }
 plan.Summary.Moved=len(plan.Changes)
 return plan,nil
}

func remapOldModID(old string,changes []agentMove)string {
 best:=old
 bestLen:=0
 for _,m:=range changes{
  if old==m.OldID||strings.HasPrefix(old,m.OldID+"/"){
   if len(m.OldID)>bestLen{
    bestLen=len(m.OldID)
    best=m.NewID+strings.TrimPrefix(old,m.OldID)
   }
  }
 }
 return best
}

func copyAgentConfig(cfg Config,changes []agentMove)Config {
 // Work only on detached maps; a failed filesystem operation must not
 // corrupt the live configuration or any saved profiles.
 raw,_:=json.Marshal(cfg)
 var next Config
 _=json.Unmarshal(raw,&next)
 if next.Tags==nil{next.Tags=map[string]string{}}
 if next.OutfitTags==nil{next.OutfitTags=map[string]string{}}
 for _,field:=range []struct{name string; values map[string]string}{
  {"tags",next.Tags},{"outfits",next.OutfitTags},
 } {
  _=field.name
  update:=map[string]string{}
  for key,value:=range field.values{
   update[remapOldModID(key,changes)]=value
  }
  if field.name=="tags"{next.Tags=update}else{next.OutfitTags=update}
 }
 for _,m:=range changes{
  // Even when an old folder name was just "body", its new agent is
  // retained after the path moves.
  next.Tags[m.NewID]=m.Agent
 }
 for name,ids:=range next.Profiles{
  unique:=map[string]bool{}
  moved:=make([]string,0,len(ids))
  for _,id:=range ids{
   id=remapOldModID(id,changes)
   if !unique[id]{unique[id]=true;moved=append(moved,id)}
  }
  next.Profiles[name]=moved
 }
 return next
}

func (a *App) applyAgentOrganization(plan agentOrganizePlan)(OrganizeResult,error){
 summary:=plan.Summary
 if len(plan.Changes)==0{return summary,nil}
 // Back up existing config before touching anything. Never overwrite
 // previous backups, even when users repeat the organizer.
 if before,err:=os.ReadFile(a.configFile);err==nil{
  name:=a.configFile+".before-agent-folders-"+time.Now().Format("20060102-150405.000000000")+".bak"
  if err:=os.WriteFile(name,before,0600);err!=nil{return summary,fmt.Errorf("Falha ao criar backup de configuração: %w",err)}
 }else if !os.IsNotExist(err){return summary,err}
 oldCfg:=a.config
 newCfg:=copyAgentConfig(oldCfg,plan.Changes)
 done:=[]agentMove{}
 madeDirs:=[]string{}
 rollback:=func(){
  for i:=len(done)-1;i>=0;i--{_ = os.Rename(done[i].To,done[i].From)}
  for i:=len(madeDirs)-1;i>=0;i--{_ = os.Remove(madeDirs[i])}
  a.config=oldCfg
 }
 for _,m:=range plan.Changes{
  dir:=filepath.Dir(m.To)
  if st,err:=os.Lstat(dir);os.IsNotExist(err){
   if err:=os.Mkdir(dir,0755);err!=nil{rollback();return summary,fmt.Errorf("Falha ao criar pasta %s: %w",m.Agent,err)}
   madeDirs=append(madeDirs,dir)
  }else if err!=nil||!st.IsDir()||st.Mode()&os.ModeSymlink!=0{
   rollback();return summary,fmt.Errorf("Destino do agente %s inválido",m.Agent)
  }
  if _,err:=os.Lstat(m.To);!os.IsNotExist(err){
   rollback();return summary,fmt.Errorf("Destino já ocupado: %s",m.To)
  }
  if err:=os.Rename(m.From,m.To);err!=nil{
   rollback()
   return summary,fmt.Errorf("Falha ao mover %s: %w (movimentos anteriores revertidos)",m.OldID,err)
  }
  done=append(done,m)
 }
 a.config=newCfg
 if err:=a.save();err!=nil{
  rollback()
  return summary,fmt.Errorf("Não foi possível salvar os novos caminhos: %w (movimentos revertidos)",err)
 }
 for _,dir:=range madeDirs{summary.Created=append(summary.Created,filepath.Base(dir))}
 return summary,nil
}

func (a *App) organizeAgentFolders(onlyInside string)(OrganizeResult,error){
 plan,err:=a.planAgentOrganization(onlyInside)
 if err!=nil{return OrganizeResult{},err}
 return a.applyAgentOrganization(plan)
}
