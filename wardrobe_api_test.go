package main
import(
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func TestEnableSkinInLibraryDisablesSameAgentSiblings(t *testing.T){
 root:=makeMods(t)
 addTestMod(t,root,"Roxy Outfit One","[TextureOverride]\nhash=12345678\n")
 addTestMod(t,root,"DISABLED_Roxy Outfit Two","[TextureOverride]\nhash=12345679\n")
 addTestMod(t,root,"Rina Dress","[TextureOverride]\nhash=1234567a\n")
 app:=&App{token:"test",configFile:filepath.Join(t.TempDir(),"config.json"),
 config:Config{ModDir:root,Tags:map[string]string{},Profiles:map[string][]string{}}}
 req:=httptest.NewRequest("POST","/api/toggle",strings.NewReader(`{"id":"Roxy Outfit Two","enabled":true}`))
 req.Header.Set("X-Desk-Token","test")
 rec:=httptest.NewRecorder();app.api(rec,req)
 if rec.Code!=200{t.Fatalf("%d %s",rec.Code,rec.Body.String())}
 for _,name:=range []string{"DISABLED_Roxy Outfit One","Roxy Outfit Two","Rina Dress"}{
  if _,err:=os.Stat(filepath.Join(root,name,"mod.ini"));err!=nil{t.Fatalf("bad active set: %s %v",name,err)}
 }
}
func TestApplyOldProfileWithMultipleSkinsIsRejectedSafely(t *testing.T){
 root:=makeMods(t)
 addTestMod(t,root,"Roxy Outfit One","hash=12345678")
 addTestMod(t,root,"DISABLED_Roxy Outfit Two","hash=12345679")
 app:=&App{token:"test",configFile:filepath.Join(t.TempDir(),"config.json"),
 config:Config{ModDir:root,Tags:map[string]string{},Profiles:map[string][]string{"Bad":{"Roxy Outfit One","Roxy Outfit Two"}}}}
 req:=httptest.NewRequest("POST","/api/apply-profile",strings.NewReader(`{"name":"Bad"}`))
 req.Header.Set("X-Desk-Token","test");rec:=httptest.NewRecorder();app.api(rec,req)
 if rec.Code!=409{t.Fatalf("expected profile conflict, got %d %s",rec.Code,rec.Body.String())}
 if _,err:=os.Stat(filepath.Join(root,"Roxy Outfit One","mod.ini"));err!=nil{t.Fatal(err)}
 if _,err:=os.Stat(filepath.Join(root,"DISABLED_Roxy Outfit Two","mod.ini"));err!=nil{t.Fatal(err)}
}
