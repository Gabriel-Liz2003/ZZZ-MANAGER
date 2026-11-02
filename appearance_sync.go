package main

import (
 "bufio"
 "bytes"
 "errors"
 "fmt"
 "io/fs"
 "os"
 "path/filepath"
 "regexp"
 "strconv"
 "strings"
 "time"
 "unicode/utf8"
)

// Match only numerical constants explicitly declared persistent in [Constants].
// Never rewrite unrelated mod definitions, keybinds, or resource paths.
var editablePersistentConstant=regexp.MustCompile(`(?i)^(\s*global\s+persist\s+\$)([a-z_][a-z0-9_]*)(\s*=\s*)([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][-+]?[0-9]+)?)(\s*(?:[;#].*)?)$`)

func runtimePersistentValues(modsRoot string)(map[string]string,error){
 path:=filepath.Join(filepath.Dir(modsRoot),"d3dx_user.ini")
 stat,err:=os.Stat(path)
 if err!=nil{return nil,errors.New("d3dx_user.ini não encontrado ao lado da pasta Mods; configure o caminho correto do ZZMI")}
 if stat.Size()>4<<20{return nil,errors.New("d3dx_user.ini muito grande")}
 if time.Since(stat.ModTime())>30*time.Minute{return nil,errors.New("d3dx_user.ini não foi atualizado recentemente; volte ao jogo, ajuste a skin e pressione F10 antes de salvar")}
 data,err:=os.ReadFile(path);if err!=nil{return nil,err}
 result:=map[string]string{}
 scanner:=bufio.NewScanner(bytes.NewReader(data))
 scanner.Buffer(make([]byte,64*1024),4<<20)
 for scanner.Scan(){
  line:=strings.TrimSpace(scanner.Text())
  if line==""||strings.HasPrefix(line,";")||strings.HasPrefix(line,"#"){continue}
  p:=strings.IndexByte(line,'=')
  if p<0{continue}
  key:=strings.ToLower(strings.TrimSpace(line[:p]))
  if !strings.HasPrefix(key,"$\\mods\\"){continue}
  key=strings.ReplaceAll(strings.TrimPrefix(key,"$\\mods\\"),"\\","/")
  val:=strings.TrimSpace(stripINIComment(line[p+1:]))
  if _,err:=strconv.ParseFloat(val,64);err!=nil{continue}
  result[key]=val
 }
 return result,scanner.Err()
}

func syncModPersistentDefaults(modsRoot string,m *Mod)(int,error){
 if !m.Enabled{return 0,errors.New("Ative a skin antes de salvar a variação")}
 if m.InheritedDisabled{return 0,errors.New("Skin dentro de pasta desativada")}
 runtime,err:=runtimePersistentValues(modsRoot)
 if err!=nil{return 0,err}
 if len(runtime)==0{return 0,errors.New("Nenhuma variável persistente de mods encontrada no d3dx_user.ini")}
 type edit struct{path string;before,after []byte}
 edits:=[]edit{}
 matched:=0
 err=filepath.WalkDir(m.path,func(path string,d fs.DirEntry,walkErr error)error{
  if walkErr!=nil{return walkErr}
  if d.IsDir(){
   if path!=m.path && (strings.HasPrefix(d.Name(),".")||nestedModPartDisabled(m.path,filepath.Join(path,"dummy.ini"))){return filepath.SkipDir}
   return nil
  }
  if !strings.EqualFold(filepath.Ext(d.Name()),".ini"){return nil}
  info,err:=d.Info();if err!=nil{return err}
  if info.Size()>maxHotkeyINIBytes{return nil}
  // Reject symlinks to avoid overwriting anything outside Mods.
  if info.Mode()&os.ModeSymlink!=0{return errors.New("Não é permitido alterar .ini em link simbólico")}
  data,err:=os.ReadFile(path);if err!=nil{return err}
  if !utf8.Valid(data){return nil} // Only UTF-8 files can be safely edited here.
  rel,err:=filepath.Rel(modsRoot,path);if err!=nil{return err}
  prefix:=strings.ToLower(filepath.ToSlash(rel))+"/"
  lines:=strings.SplitAfter(string(data),"\n")
  inConstants:=false;changed:=false
  for i,line:=range lines{
   trimmed:=strings.TrimSpace(strings.TrimPrefix(line,"\ufeff"))
   if strings.HasPrefix(trimmed,"["){inConstants=strings.EqualFold(trimmed,"[Constants]");continue}
   if !inConstants||strings.HasPrefix(trimmed,";")||strings.HasPrefix(trimmed,"#"){continue}
   clean:=strings.TrimRight(line,"\r\n")
   match:=editablePersistentConstant.FindStringSubmatch(clean)
   if len(match)==0{continue}
   val,ok:=runtime[prefix+strings.ToLower(match[2])]
   if !ok{continue}
   matched++
   if val!=match[4]{
    tail:=strings.TrimPrefix(line,clean)
    lines[i]=match[1]+match[2]+match[3]+val+match[5]+tail
    changed=true
   }
  }
  if changed{edits=append(edits,edit{path:path,before:data,after:[]byte(strings.Join(lines,""))})}
  return nil
 })
 if err!=nil{return 0,err}
 if matched==0{return 0,errors.New("Nenhuma variável compatível encontrada. O mod pode não declarar global persist, usar namespace personalizado ou não ter sido salvo com F10. Nenhum arquivo foi alterado.")}
 // Create backups first (without overwriting existing backup copies).
 for _,e:=range edits{
  backup:=e.path+".zzzmanager.bak"
  if _,err:=os.Stat(backup);os.IsNotExist(err){
   if err:=os.WriteFile(backup,e.before,0600);err!=nil{return 0,fmt.Errorf("Não foi possível criar backup de %s: %w",filepath.Base(e.path),err)}
  }else if err!=nil{return 0,err}
 }
 done:=[]edit{}
 for _,e:=range edits{
  if err:=os.WriteFile(e.path,e.after,0644);err!=nil{
   for i:=len(done)-1;i>=0;i--{_ = os.WriteFile(done[i].path,done[i].before,0644)}
   return 0,fmt.Errorf("Falha ao salvar variação: %w",err)
  }
  done=append(done,e)
 }
 return matched,nil
}
