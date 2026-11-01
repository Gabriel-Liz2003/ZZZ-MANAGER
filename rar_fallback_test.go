package main

import (
    "bytes"
    "errors"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/nwaples/rardecode/v2"
)

func TestValidateRARWindowsTarListing(t *testing.T) {
    names := "Roxy Outfit/config.ini\nRoxy Outfit/toggles/hair.ini\nRoxy Outfit/textures/body.dds\nRoxy Outfit/resources/\n"
    modes := "-rw-r--r--  0 0 0 10 date config.ini\n-rw-r--r--  0 0 0 12 date hair.ini\n-rw-r--r--  0 0 0 20 date body.dds\ndrwxr-xr-x  0 0 0 0 date resources\n"
    actual,err := validateRarTarListing(names,modes)
    if err!=nil{t.Fatal(err)}
    if len(actual)!=3 || actual[0]!="Roxy Outfit/config.ini" {t.Fatalf("unexpected list: %#v",actual)}
}

func TestRejectDangerousRARListing(t *testing.T) {
    cases:=[]struct{name string; mode byte}{
        {"../escape.ini", '-'},
        {"/abs/path.ini", '-'},
        {"Roxy/../../escape.ini", '-'},
        {"Roxy/setup.exe", '-'},
        {"Roxy/CON.ini", '-'},
        {"Roxy/link.ini", 'l'},
        {"Roxy/hardlink.ini", 'h'},
        {"C:\\Windows\\evil.ini", '-'},
        {"Roxy/./mod.ini", '-'},
    }
    for _,tc:=range cases{
        info:=string(tc.mode)+"rw-r--r--  0 0 0 10 date "+tc.name+"\n"
        if _,err:=validateRarTarListing(tc.name+"\n",info);err==nil{t.Errorf("accepted unsafe entry %q mode %q",tc.name,tc.mode)}
    }
}

func TestRejectDuplicateWindowsRARNames(t *testing.T) {
    _,err:=validateRarTarListing("Roxy/mod.ini\nroxy/MOD.ini\n","-rw-r--r-- first\n-rw-r--r-- second\n")
    if err==nil{t.Fatal("duplicate case insensitive names were allowed")}
}

func TestRARSignature(t *testing.T) {
    root:=t.TempDir()
    good4:=filepath.Join(root,"rar4.rar")
    good5:=filepath.Join(root,"rar5.rar")
    fake:=filepath.Join(root,"fake.rar")
    if err:=os.WriteFile(good4,[]byte{0x52,0x61,0x72,0x21,0x1a,0x07,0x00},0600);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(good5,[]byte{0x52,0x61,0x72,0x21,0x1a,0x07,0x01,0x00},0600);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(fake,[]byte("This is not a RAR archive"),0600);err!=nil{t.Fatal(err)}
    if err:=verifyRARSignature(good4);err!=nil{t.Fatal(err)}
    if err:=verifyRARSignature(good5);err!=nil{t.Fatal(err)}
    if err:=verifyRARSignature(fake);err==nil{t.Fatal("fake archive accepted")}
}

func TestMalformedRARDoesNotCreatePartialMods(t *testing.T) {
    root:=makeMods(t)
    // Starts with RAR5 magic but has no archive payload.
    _,err:=importArchive(root,"Fake.rar",bytes.NewReader([]byte{0x52,0x61,0x72,0x21,0x1a,0x07,0x01,0x00}))
    if err==nil{t.Fatal("expected archive decode error")}
    files,err:=os.ReadDir(root)
    if err!=nil{t.Fatal(err)}
    if len(files)!=0{t.Fatalf("failed archive should leave no mod folders: %#v",files)}
}

func TestRARFallbackOnlyForRecoverableErrors(t *testing.T) {
    if !shouldTryRARFallback(rardecode.ErrShortFile){t.Fatal("short RAR should try Windows decoder")}
    if !shouldTryRARFallback(errors.New("rardecode: unsupported RAR compression")){t.Fatal("decode failures should be retried")}
    if shouldTryRARFallback(errors.New("RAR contém link ou arquivo especial")){t.Fatal("dangerous links must never be retried")}
    if shouldTryRARFallback(errors.New("Caminho inseguro: ../evil.ini")){t.Fatal("unsafe path must never be retried")}
}

func TestRARFriendlyErrorMessage(t *testing.T) {
    err:=importRARError(rardecode.ErrShortFile,errors.New("tar does not support this compression"))
    msg:=err.Error()
    if !strings.Contains(msg,"Instalar pasta") || !strings.Contains(msg,"7-Zip") || !strings.Contains(msg,"decoded file too short") {
        t.Fatal(msg)
    }
}
