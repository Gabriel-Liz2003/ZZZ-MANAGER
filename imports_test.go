package main

import (
 "bytes"
 "mime/multipart"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func folderUpload(t *testing.T, files map[string]string) *multipart.Reader {
 t.Helper()
 var b bytes.Buffer
 w:=multipart.NewWriter(&b)
 for path,body:=range files {
  if err:=w.WriteField("path",path);err!=nil{t.Fatal(err)}
  part,err:=w.CreateFormFile("file",filepath.Base(path))
  if err!=nil{t.Fatal(err)}
  if _,err=part.Write([]byte(body));err!=nil{t.Fatal(err)}
 }
 if err:=w.Close();err!=nil{t.Fatal(err)}
 return multipart.NewReader(&b,w.Boundary())
}

func TestFolderUploadAndNoOverwrite(t *testing.T){
 root:=makeMods(t)
 files:=map[string]string{"Casual/config.ini":"[TextureOverride]\nhash=12345678","Casual/preview.png":"picture"}
 n,err:=importFolder(root,"Miyabi Casual",folderUpload(t,files))
 if err!=nil{t.Fatal(err)}
 if n!="Miyabi Casual"{t.Fatal(n)}
 if _,err=os.Stat(filepath.Join(root,n,"Casual","config.ini"));err!=nil{t.Fatal(err)}
 n,err=importFolder(root,"Miyabi Casual",folderUpload(t,files))
 if err!=nil{t.Fatal(err)}
 if n!="Miyabi Casual (2)"{t.Fatal(n)}
 mods,_,err:=scanMods(root,nil)
 if err!=nil || len(mods)!=2{t.Fatalf("mods %d, err %v",len(mods),err)}
}

func TestFolderUploadRejectsDangerousFiles(t *testing.T){
 for _,file:=range []string{"../escape.ini","C:/Windows/system.ini","Foo/../../evil.ini","Install.bat","Any/CON.ini","Any/script.exe"}{
  root:=makeMods(t)
  _,err:=importFolder(root,"Bad Folder",folderUpload(t,map[string]string{file:"hash=abcdef12"}))
  if err==nil{t.Errorf("not rejected: %s",file)}
  entries,_:=os.ReadDir(root)
  if len(entries)!=0{t.Errorf("import left files after error: %s",file)}
 }
}

func TestFolderNeedsIniAndZeroByteFiles(t *testing.T){
 root:=makeMods(t)
 _,err:=importFolder(root,"No ini",folderUpload(t,map[string]string{"a.txt":"hello"}))
 if err==nil || !strings.Contains(err.Error(),"ini"){t.Fatal(err)}
 n,err:=importFolder(root,"Zero",folderUpload(t,map[string]string{"one/mod.ini":""}))
 if err!=nil{t.Fatal(err)}
 b,err:=os.ReadFile(filepath.Join(root,n,"one","mod.ini"))
 if err!=nil || len(b)!=0{t.Fatal(err,len(b))}
}

func TestArchiveRejectsUnknownExtension(t *testing.T){
 root:=makeMods(t)
 if _,err:=importArchive(root,"setup.exe",strings.NewReader("contents"));err==nil{t.Fatal("expected error")}
}
