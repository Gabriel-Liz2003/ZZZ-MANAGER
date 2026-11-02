package main
import (
 "crypto/rand"
 "errors"
 "fmt"
 "math/big"
 "sort"
 "strings"
)
// A skin choice may be one installed mod or the original (empty modId).
type WardrobeChoice struct {
 Agent string `json:"agent"`
 ModID string `json:"modId"`
 Name string `json:"name"`
}
type wardrobeChange struct {mod *Mod; enabled bool}
func wardrobeGroup(mods []*Mod, agent string) []*Mod {
 result:=[]*Mod{}
 for _,m:=range mods{if strings.EqualFold(m.Character,agent){result=append(result,m)}}
 return result
}
func knownWardrobeAgent(agent string) bool{
 agent=strings.TrimSpace(agent)
 if agent==""||strings.EqualFold(agent,"Sem categoria"){return false}
 for _,name:=range characterNames{if strings.EqualFold(name,agent){return true}}
 return false
}
func validateWardrobeGroup(group []*Mod)error{
 if len(group)==0{return errors.New("Agente sem skins instaladas")}
 ids:=map[string]bool{}
 for _,m:=range group{
  id:=strings.ToLower(m.ID)
  if ids[id]{return fmt.Errorf("Há pastas duplicadas para o mod %s; renomeie uma antes de alternar",m.ID)}
  ids[id]=true
  if m.InheritedDisabled{return fmt.Errorf("O mod %s está dentro de uma pasta superior desativada",m.Name)}
 }
 return nil
}
func selectWardrobe(mods []*Mod, agent,id string)([]wardrobeChange,WardrobeChoice,error){
 if !knownWardrobeAgent(agent){return nil,WardrobeChoice{},errors.New("Agente inválido ou não categorizado")}
 group:=wardrobeGroup(mods,agent)
 if err:=validateWardrobeGroup(group);err!=nil{return nil,WardrobeChoice{},err}
 choice:=WardrobeChoice{Agent:agent,Name:"Aparência original"}
 if id!=""{
  found:=false
  for _,m:=range group{if m.ID==id{found=true;choice.ModID=m.ID;choice.Name=m.Name;break}}
  if !found{return nil,WardrobeChoice{},errors.New("Skin não pertence a este agente")}
 }
 changes:=[]wardrobeChange{}
 for _,m:=range group{if m.Enabled&&m.ID!=id{changes=append(changes,wardrobeChange{m,false})}}
 for _,m:=range group{if !m.Enabled&&m.ID==id{changes=append(changes,wardrobeChange{m,true})}}
 return changes,choice,nil
}
func cryptoChoose(max int)(int,error){
 if max<=0{return 0,errors.New("Sem opções")}
 n,err:=rand.Int(rand.Reader,big.NewInt(int64(max)));if err!=nil{return 0,err}
 return int(n.Int64()),nil
}
// Randomize only agents which already have at least one enabled skin.
// Exclude the sole current skin so every eligible agent visibly changes.
func planRandomWardrobe(mods []*Mod,choose func(int)(int,error))([]wardrobeChange,[]WardrobeChoice,error){
 agentSet:=map[string]bool{}
 for _,m:=range mods{if m.Enabled&&knownWardrobeAgent(m.Character){agentSet[m.Character]=true}}
 agents:=[]string{}
 for name:=range agentSet{agents=append(agents,name)}
 sort.Strings(agents)
 if len(agents)==0{return nil,nil,errors.New("Nenhum agente com skin ativa. Ative e classifique um mod primeiro.")}
 changes:=[]wardrobeChange{}
 choices:=[]WardrobeChoice{}
 for _,agent:=range agents{
  group:=wardrobeGroup(mods,agent)
  if err:=validateWardrobeGroup(group);err!=nil{return nil,nil,fmt.Errorf("%s: %w",agent,err)}
  active:=[]string{}
  for _,m:=range group{if m.Enabled{active=append(active,m.ID)}}
  options:=[]string{""}
  for _,m:=range group{if len(active)==1&&m.ID==active[0]{continue};options=append(options,m.ID)}
  i,err:=choose(len(options));if err!=nil{return nil,nil,err}
  if i<0||i>=len(options){return nil,nil,errors.New("Sorteio fora dos limites")}
  plan,pick,err:=selectWardrobe(mods,agent,options[i])
  if err!=nil{return nil,nil,err}
  changes=append(changes,plan...);choices=append(choices,pick)
 }
 return changes,choices,nil
}
func applyWardrobeChanges(changes []wardrobeChange)error{
 completed:=[]wardrobeChange{}
 for _,change:=range changes{
  if err:=toggleMod(change.mod,change.enabled);err!=nil{
   failed:=false
   for i:=len(completed)-1;i>=0;i--{if toggleMod(completed[i].mod,!completed[i].enabled)!=nil{failed=true}}
   if failed{return fmt.Errorf("Falha ao alternar skins: %v. Restauração incompleta, confira as pastas.",err)}
   return fmt.Errorf("Falha ao alternar skins: %v. Alterações revertidas.",err)
  }
  completed=append(completed,change)
 }
 return nil
}
