package main

import (
 "bufio"
 "os"
 "path/filepath"
 "regexp"
 "strings"
)

var persistentVariablePattern=regexp.MustCompile(`(?i)^\s*(?:global\s+persist|persist\s+global)\s+\$[a-z_][a-z0-9_]*\b`)

// countModPersistDefinitions reports which toggles explicitly request
// 3DMigoto state persistence. It never edits mods or promises that a
// particular in-game key is persistent. Unknown/passthrough INI syntax
// remains untouched.
func countModPersistDefinitions(path string) int {
 info,err:=os.Stat(path);if err!=nil||info.Size()>maxHotkeyINIBytes {return 0}
 raw,err:=os.ReadFile(path);if err!=nil{return 0}
 text:=iniHotkeyText(raw)
 scanner:=bufio.NewScanner(strings.NewReader(text))
 scanner.Buffer(make([]byte,64*1024),maxHotkeyINIBytes+1)
 inside:=false
 count:=0
 for scanner.Scan(){
  line:=strings.TrimSpace(scanner.Text())
  if strings.HasPrefix(line,";")||strings.HasPrefix(line,"#")||line==""{continue}
  if strings.HasPrefix(line,"["){
   inside=strings.EqualFold(line,"[Constants]")
   continue
  }
  if inside && persistentVariablePattern.MatchString(line){count++}
 }
 return count
}

func countPersistInModDir(root string) int {
 count:=0
 _=filepath.WalkDir(root,func(path string, entry os.DirEntry, err error) error {
  if err!=nil{return nil}
  if entry.IsDir(){
   if path!=root&&strings.HasPrefix(entry.Name(),"."){return filepath.SkipDir}
   return nil
  }
  if strings.EqualFold(filepath.Ext(entry.Name()),".ini")&&!nestedModPartDisabled(root,path){
   count+=countModPersistDefinitions(path)
  }
  return nil
 })
 return count
}
