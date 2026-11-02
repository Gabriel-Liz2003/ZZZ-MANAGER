package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscordCatalogAttachmentValidation(t *testing.T) {
	good := []string{
		"https://cdn.discordapp.com/attachments/123/456/Remielle_Police.zip?ex=123&is=456&hm=789",
		"https://cdn.discord.com/attachments/123/456/Skin.rar",
		"https://media.discordapp.net/attachments/123/456/Skin.7z",
		"https://cdn.discordapp.com/ephemeral-attachments/123/456/Skin.zip",
	}
	for _, link := range good {
		if _, err := isDiscordCDNURL(link, true); err != nil {
			t.Errorf("expected archive URL %q to be accepted: %v", link, err)
		}
	}
	bad := []string{
		"https://discord.com/channels/12/34/56",
		"http://cdn.discordapp.com/attachments/12/34/Skin.zip",
		"https://cdn.discordapp.com.evil.example/attachments/12/34/Skin.zip",
		"https://cdn.discordapp.com:8443/attachments/12/34/Skin.zip",
		"https://cdn.discordapp.com/attachments/12/34/evil.exe",
		"https://127.0.0.1/attachments/12/34/Skin.zip",
		"https://cdn.discordapp.com/avatars/12/34/Skin.zip",
		"file:///C:/fake/Skin.zip",
	}
	for _, link := range bad {
		if _, err := isDiscordCDNURL(link, true); err == nil {
			t.Errorf("unsafe URL accepted: %q", link)
		}
	}
	for _, pair := range [][2]string{
		{"https://cdn.discordapp.com/attachments/12/34/preview.png", "ok"},
		{"https://cdn.discordapp.com/attachments/12/34/mod.zip", "bad"},
	} {
		_, err := isDiscordCDNURL(pair[0], false)
		if (err == nil) != (pair[1] == "ok") {
			t.Fatalf("wrong image URL validation: %q %v", pair[0], err)
		}
	}
}

type testDiscordRoundTripper func(*http.Request) (*http.Response, error)

func (f testDiscordRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDownloadDiscordArchiveUsesExistingInstaller(t *testing.T) {
	root := makeMods(t)
	zip := makeZIP(t, map[string]string{
		"Remielle Costume/mod.ini": "[KeySwap]\nkey = l\n[TextureOverrideA]\nhash = deadbeef\n",
		"Remielle Costume/preview.png": "image",
	})
	client := &http.Client{Transport: testDiscordRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.URL.Hostname() != "cdn.discordapp.com" {
			t.Fatalf("unexpected hostname %s", req.URL.Hostname())
		}
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(bytes.NewReader(zip)),
			ContentLength: int64(len(zip)),
			Header: make(http.Header),
			Request: req,
		}, nil
	})}
	name, err := downloadDiscordArchive(context.Background(), root,
		"https://cdn.discordapp.com/attachments/111/222/Remielle_Police.zip?ex=123", client)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, name, "Remielle Costume", "mod.ini")); err != nil {
		t.Fatalf("ZIP was not extracted: %v", err)
	}
	a := &App{configFile:filepath.Join(t.TempDir(),"config.json"),config:Config{ModDir:root,Tags:map[string]string{}}}
	count, err := a.tagNewDiscordArchive(name, "Remielle")
	if err != nil || count != 1 {
		t.Fatalf("automatic agent organization failed: %v (count=%d)", err, count)
	}
	mods, _, err := scanMods(root, a.config.Tags)
	if err != nil || len(mods) != 1 || mods[0].Character != "Remielle" {
		t.Fatalf("wrong organized mods: %v, %+v", err, mods)
	}
	if _, err := os.Stat(filepath.Join(root,"Remielle","Remielle Costume","mod.ini")); err != nil {
		t.Fatalf("the mod must be organized beneath its agent: %v",err)
	}
}

func TestDiscordCatalogRejectsExpiredAttachment(t *testing.T) {
	root := makeMods(t)
	client := &http.Client{Transport:testDiscordRoundTripper(func(req *http.Request) (*http.Response,error){
		return &http.Response{StatusCode:403,Body:io.NopCloser(strings.NewReader("")),Header:make(http.Header),Request:req},nil
	})}
	_,err:=downloadDiscordArchive(context.Background(),root,"https://cdn.discordapp.com/attachments/12/34/file.zip",client)
	if err==nil||!strings.Contains(err.Error(),"expirou"){t.Fatalf("expected useful expired link error: %v",err)}
	mods,_,err:=scanMods(root,nil)
	if err!=nil||len(mods)!=0{t.Fatalf("failed download must leave Mods clean: %v, %+v",err,mods)}
}

func TestDiscordCatalogAddAndRemoveAPI(t *testing.T) {
	root:=makeMods(t)
	a:=&App{token:"secret",configFile:filepath.Join(t.TempDir(),"config.json"),config:Config{ModDir:root,Tags:map[string]string{}}}
	apiPost:=func(route,body string)*httptest.ResponseRecorder{
		rec:=httptest.NewRecorder()
		req:=httptest.NewRequest(http.MethodPost,"/api/"+route,strings.NewReader(body))
		req.Header.Set("X-Desk-Token","secret")
		a.api(rec,req)
		return rec
	}
	payload:=`{"name":"Remielle Police","agent":"Remielle","downloadUrl":"https://cdn.discordapp.com/attachments/123/456/Remielle_Police.zip?ex=123","messageUrl":"https://discord.com/channels/111/222/333"}`
	if rec:=apiPost("discord/add",payload);rec.Code!=200{t.Fatalf("add: %d %s",rec.Code,rec.Body.String())}
	if len(a.config.DiscordMods)!=1||a.config.DiscordMods[0].ID==""{t.Fatalf("bookmark not persisted: %+v",a.config.DiscordMods)}
	if rec:=apiPost("discord/add",payload);rec.Code!=409{t.Fatalf("duplicate should reject: %d %s",rec.Code,rec.Body.String())}
	if rec:=apiPost("discord/add",strings.Replace(payload,"cdn.discordapp.com","evil.example",1));rec.Code!=400{t.Fatalf("untrusted host accepted: %d",rec.Code)}
	id:=a.config.DiscordMods[0].ID
	newLink:=fmt.Sprintf(`{"id":%q,"downloadUrl":"https://cdn.discordapp.com/attachments/123/456/Remielle_Police.zip?ex=456"}`,id)
	if rec:=apiPost("discord/update-link",newLink);rec.Code!=200{t.Fatalf("update link: %d %s",rec.Code,rec.Body.String())}
	if !strings.HasSuffix(a.config.DiscordMods[0].DownloadURL,"ex=456"){t.Fatal("did not update URL")}
	if rec:=apiPost("discord/remove",fmt.Sprintf(`{"id":%q}`,id));rec.Code!=200{t.Fatalf("remove: %d %s",rec.Code,rec.Body.String())}
	if len(a.config.DiscordMods)!=0 {t.Fatal("catalog entry still exists")}
}
