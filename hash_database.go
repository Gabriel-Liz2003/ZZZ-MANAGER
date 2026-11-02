package main

import (
 "bufio"
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "os"
 "path/filepath"
 "regexp"
 "sort"
 "strings"
 "time"
)

// Source: ZZZ Mod Hub's community-curated character and outfit hash catalog.
// The database is downloaded on demand and stored locally, separate from user mods.
// No imported project source code is embedded in this app.
const communityHashDatabaseURL = "https://raw.githubusercontent.com/the-sea-man/ZZZ-Mod-Hub/main/src/characters.json"
const hashDatabaseMaxBytes = 4 << 20
const hashDatabaseFilename = "model_hash_database.json"

type hashDatabase struct {
 Characters []hashCharacter `json:"characters"`
}
type hashCharacter struct {
 Name string `json:"name"`
 Aliases []string `json:"aliases"`
 BaseSkinID string `json:"base_skin_id"`
 Type string `json:"type"`
 Skins []hashSkin `json:"skins"`
}
type hashSkin struct {
 ID string `json:"id"`
 Name string `json:"name"`
 Components map[string]hashComponent `json:"components"`
}
type hashComponent struct {
 DrawVB string `json:"draw_vb"`
 PositionVB string `json:"position_vb"`
 BlendVB string `json:"blend_vb"`
 TexcoordVB string `json:"texcoord_vb"`
 IB string `json:"ib"`
}
type hashOwner struct{ Agent, OutfitID string }
type hashCatalog struct{
 Owners map[string][]hashOwner
 Outfits []OfficialOutfit
 HashCount int
}
type hashDetection struct{
 Agent string
 OutfitID string
 Confidence string
 Evidence int
 Status string
}
var modelHashLineRE = regexp.MustCompile(`(?i)^hash\s*=\s*([a-f0-9]{8})(?:\s|$|[;#])`)

func parseCommunityHashDatabase(data []byte)(*hashDatabase,error){
 if len(data)>hashDatabaseMaxBytes{return nil,errors.New("Banco de hashes ultrapassa 4 MB")}
 var db hashDatabase
 if err:=json.Unmarshal(data,&db);err!=nil{return nil,fmt.Errorf("JSON de hashes inválido: %w",err)}
 if len(db.Characters)<35{return nil,errors.New("Base de hashes parece incompleta (menos de 35 personagens)")}
 count:=0
 for _,c:=range db.Characters{for _,s:=range c.Skins{count+=len(s.Components)}}
 if count<100{return nil,errors.New("Base de hashes não tem componentes de modelo suficientes")}
 return &db,nil
}
func downloadCommunityHashDatabase(ctx context.Context)([]byte,*hashDatabase,error){
 client:=&http.Client{Timeout:12*time.Second}
 req,err:=http.NewRequestWithContext(ctx,http.MethodGet,communityHashDatabaseURL,nil)
 if err!=nil{return nil,nil,err}
 req.Header.Set("User-Agent","ZZZ-MANAGER/3.6 community-hash-sync")
 res,err:=client.Do(req)
 if err!=nil{return nil,nil,fmt.Errorf("Falha ao baixar a base de hashes: %w",err)}
 defer res.Body.Close()
 if res.StatusCode!=http.StatusOK{return nil,nil,fmt.Errorf("Servidor retornou HTTP %d ao baixar os hashes",res.StatusCode)}
 raw,err:=io.ReadAll(io.LimitReader(res.Body,hashDatabaseMaxBytes+1))
 if err!=nil{return nil,nil,err}
 db,err:=parseCommunityHashDatabase(raw)
 return raw,db,err
}
func readCommunityHashDatabase(configFile string)*hashDatabase{
 raw,err:=os.ReadFile(filepath.Join(filepath.Dir(configFile),hashDatabaseFilename))
 if err!=nil{return nil}
 db,err:=parseCommunityHashDatabase(raw)
 if err!=nil{return nil}
 return db
}
func persistCommunityHashDatabase(configFile string,raw []byte)error{
 dir:=filepath.Dir(configFile)
 file:=filepath.Join(dir,hashDatabaseFilename)
 temp,err:=os.CreateTemp(dir,".hashdb-*.tmp")
 if err!=nil{return err}
 defer os.Remove(temp.Name())
 if _,err=temp.Write(raw);err!=nil{temp.Close();return err}
 if err=temp.Close();err!=nil{return err}
 if err:=os.Rename(temp.Name(),file);err!=nil{
  // Windows does not replace destination with Rename.
  if _,statErr:=os.Stat(file);statErr==nil{
   // File is only the local cache; preserving it if replace fails is safe.
   if err=os.Remove(file);err!=nil{return err}
   return os.Rename(temp.Name(),file)
  }
  return err
 }
 return nil
}
func simpleName(name string)string{
 var b strings.Builder
 for _,r:=range strings.ToLower(name){
  if r>='a'&&r<='z'||r>='0'&&r<='9'{b.WriteRune(r)}
 }
 return b.String()
}
func mapCommunityAgent(c hashCharacter)string{
 candidates:=append([]string{c.Name},c.Aliases...)
 for _,name:=range candidates{
  v:=simpleName(name)
  for _,candidate:=range characterNames{
   if simpleName(candidate)==v{return candidate}
  }
 }
 return ""
}
func slugHashOutfit(name string)string{
 var b strings.Builder
 dash:=false
 for _,r:=range strings.ToLower(name){
  if r>='a'&&r<='z'||r>='0'&&r<='9'{
   if dash&&b.Len()>0{b.WriteByte('-')}
   b.WriteRune(r);dash=false
  }else if r=='''||r=='’'{continue}else{dash=true}
 }
 return b.String()
}
func outfitForCommunitySkin(agent string,c hashCharacter,s hashSkin)(OfficialOutfit,bool){
 if s.ID==c.BaseSkinID{return OfficialOutfit{Agent:agent,ID:"base",Name:s.Name,Base:true},true}
 // Prefer stable outfit IDs already used by the manager.
 for _,o:=range specialOfficialOutfits{
  if !strings.EqualFold(o.Agent,agent){continue}
  if simpleName(o.Name)==simpleName(s.Name) || simpleName(o.ID)==simpleName(s.ID){
   return o,true
  }
 }
 if s.Name==""{return OfficialOutfit{},false}
 return OfficialOutfit{Agent:agent,ID:slugHashOutfit(s.Name),Name:s.Name},true
}
func catalogFromDatabase(db *hashDatabase)*hashCatalog{
 cat:=&hashCatalog{Owners:map[string][]hashOwner{},Outfits:[]OfficialOutfit{}}
 if db==nil{return cat}
 knownOutfits:=map[string]bool{}
 for _,c:=range db.Characters{
  agent:=mapCommunityAgent(c)
  if agent==""{continue}
  for _,s:=range c.Skins{
   o,ok:=outfitForCommunitySkin(agent,c,s)
   if !ok{continue}
   key:=agent+"/"+o.ID
   if !o.Base&&!knownOutfits[key]{
    knownOutfits[key]=true
    cat.Outfits=append(cat.Outfits,o)
   }
   owner:=hashOwner{Agent:agent,OutfitID:o.ID}
   for _,part:=range s.Components {
    for _,h:=range []string{part.DrawVB,part.PositionVB,part.IB,part.BlendVB,part.TexcoordVB}{
     h=strings.ToLower(strings.TrimSpace(h))
     if len(h)!=8||!modelHashLineRE.MatchString("hash="+h){continue}
     owners:=cat.Owners[h]
     found:=false
     for _,prev:=range owners{if prev==owner{found=true;break}}
     if !found{cat.Owners[h]=append(owners,owner)}
    }
   }
  }
 }
 cat.HashCount=len(cat.Owners)
 sort.Slice(cat.Outfits,func(i,j int)bool{
  if cat.Outfits[i].Agent==cat.Outfits[j].Agent{return cat.Outfits[i].Name<cat.Outfits[j].Name}
  return cat.Outfits[i].Agent<cat.Outfits[j].Agent
 })
 return cat
}
func resolvedOfficialOutfits(cat *hashCatalog)[]OfficialOutfit{
 result:=append([]OfficialOutfit{},specialOfficialOutfits...)
 if cat==nil{return result}
 known:=map[string]bool{}
 for _,o:=range result{known[o.Agent+"/"+o.ID]=true}
 for _,o:=range cat.Outfits{
  key:=o.Agent+"/"+o.ID
  if !known[key]{result=append(result,o);known[key]=true}
 }
 return result
}
// Parse only hash declarations from INI, without changing files or following
// disabled variant folders. Top-level disabled mod packages are still indexed.
func gatherModelHashes(modRoot string)map[string]bool{
 result:=map[string]bool{}
 _=filepath.WalkDir(modRoot,func(p string,d os.DirEntry,err error)error{
  if err!=nil{return nil}
  if d.IsDir(){
   if p!=modRoot&&(strings.HasPrefix(d.Name(),".")||isDisabled(d.Name())){return filepath.SkipDir}
   return nil
  }
  if !strings.EqualFold(filepath.Ext(d.Name()),".ini"){return nil}
  stat,err:=d.Info();if err!=nil||stat.Size()<1||stat.Size()>2<<20{return nil}
  data,err:=os.ReadFile(p);if err!=nil{return nil}
  scanner:=bufio.NewScanner(bytes.NewBufferString(iniHotkeyText(data)))
  scanner.Buffer(make([]byte,4096),2<<20)
  for scanner.Scan(){
   line:=strings.TrimSpace(scanner.Text())
   if line==""||strings.HasPrefix(line,";")||strings.HasPrefix(line,"#"){continue}
   match:=modelHashLineRE.FindStringSubmatch(line)
   if len(match)==2{result[strings.ToLower(match[1])]=true}
  }
  return nil
 })
 return result
}
func identifyFromModelHashes(hashes map[string]bool,cat *hashCatalog)hashDetection{
 result:=hashDetection{OutfitID:"unknown",Status:"Não identificado",Confidence:"none"}
 if cat==nil||len(cat.Owners)==0{return result}
 hits:=map[hashOwner]int{}
 shared:=0
 for h:=range hashes{
  owners:=cat.Owners[h]
  if len(owners)==1{hits[owners[0]]++}else if len(owners)>1{shared++}
 }
 if len(hits)==0{
  if shared>0{result.Status="Hashes compartilhados entre trajes"}
  return result
 }
 if len(hits)>1{
  result.Status="Ambíguo: hashes de mais de um traje"
  return result
 }
 for owner,n:=range hits{
  result.Agent=owner.Agent
  result.OutfitID=owner.OutfitID
  result.Evidence=n
  result.Confidence="medium"
  if n>=2{result.Confidence="high"}
  result.Status="Detectado por hashes"
 }
 return result
}
