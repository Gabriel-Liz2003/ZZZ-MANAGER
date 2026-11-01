package main

import (
	"archive/zip"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed ui.html portraits.js assets/portraits/*
var embedded embed.FS

const (
	appName        = "ZZZ Mod Desk"
	disabledPrefix = "DISABLED_"
	maxUpload      = int64(1024 * 1024 * 1024)
	maxExtract     = int64(2 * 1024 * 1024 * 1024)
)

var hashRE = regexp.MustCompile(`(?im)^\s*hash\s*=\s*([0-9a-f]{8,16})\s*(?:;.*)?$`)
var invalidWindowsNameRE = regexp.MustCompile(`[<>:"\\|?*\x00-\x1f]`)

type Config struct {
	ModDir     string              `json:"modDir"`
	Tags       map[string]string   `json:"tags"`
	Profiles   map[string][]string `json:"profiles"`
	Favorites  []string            `json:"favorites"`
	PhotoFiles map[string]string   `json:"photoFiles"`
}
type Mod struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Character         string `json:"character"`
	RelPath           string `json:"relPath"`
	Enabled           bool   `json:"enabled"`
	InheritedDisabled bool   `json:"inheritedDisabled"`
	IniCount          int    `json:"iniCount"`
	Preview           bool   `json:"preview"`
	LastModified      string `json:"lastModified"`
	Conflicts         int    `json:"conflicts"`
	Hotkeys           []ModHotkey `json:"hotkeys"`
	path              string
	previewPath       string
	hashes            map[string]bool
}
type Conflict struct {
	Hash     string   `json:"hash"`
	ModNames []string `json:"modNames"`
	ModIDs   []string `json:"modIds"`
}
type Profile struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
type State struct {
	Configured   bool       `json:"configured"`
	ModDir       string     `json:"modDir"`
	Mods         []*Mod     `json:"mods"`
	Profiles     []Profile  `json:"profiles"`
	Conflicts    []Conflict `json:"conflicts"`
	Installed    int        `json:"installed"`
	Active       int        `json:"active"`
	Inactive     int        `json:"inactive"`
	Favorites    []string   `json:"favorites"`
	CustomPhotos []string   `json:"customPhotos"`
}
type APIError struct {
	Error string `json:"error"`
}
type App struct {
	mu           sync.Mutex
	config       Config
	configFile   string
	token        string
	previewCache map[string]string
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	userConfig, err := os.UserConfigDir()
	if err != nil {
		userConfig = "."
	}
	dataDir := filepath.Join(userConfig, appName)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		log.Fatal(err)
	}
	lf, err := os.OpenFile(filepath.Join(dataDir, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		defer lf.Close()
		log.SetOutput(lf)
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		log.Fatal(err)
	}
	app := &App{configFile: filepath.Join(dataDir, "config.json"), token: base64.RawURLEncoding.EncodeToString(secret), previewCache: make(map[string]string)}
	app.load()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	baseURL := fmt.Sprintf("http://127.0.0.1:%d/app/%s", port, app.token)
	mux := http.NewServeMux()
	mux.HandleFunc("/app/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/app/"+app.token {
			http.NotFound(w, r)
			return
		}
		body, err := embedded.ReadFile("ui.html")
		if err != nil {
			http.Error(w, "Interface indisponível", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data: https://fastcdn.hoyoverse.com; connect-src 'self'; base-uri 'none'; form-action 'none'")
		w.Write([]byte(strings.ReplaceAll(string(body), "__APP_TOKEN__", app.token)))
	})
	mux.HandleFunc("/assets/portraits.js", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		data, err := embedded.ReadFile("portraits.js")
		if err != nil {
			http.Error(w, "Not found", 404)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/assets/portraits/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/assets/portraits/")
		// Fixed allowlist of eight names prevents arbitrary embedded file reads.
		switch name {
		case "Belle.png", "Wise.png", "Remielle.png", "Sigrid.png",
			"Claret.png", "Roxy.png", "Phoenix.png", "Severian.png":
		default:
			http.NotFound(w, r)
			return
		}
		b, err := embedded.ReadFile("assets/portraits/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if len(b) < 8 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/api/", app.api)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != fmt.Sprintf("127.0.0.1:%d", port) {
			http.Error(w, "Host não permitido", 403)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	}), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Println("Servidor:", err)
		}
	}()
	log.Println("ZZZ Mod Desk iniciado")
	if os.Getenv("ZZZ_MOD_DESK_TEST") == "1" {
		fmt.Println(baseURL)
	} else if err := openBrowser(baseURL); err != nil {
		log.Println("Browser:", err)
		fmt.Println(baseURL)
	}
	// Server runs until the user clicks "Fechar aplicativo" or Windows ends it.
	select {}
}
func openBrowser(url string) error {
	if runtime.GOOS == "windows" {
		// Start a minimal browser-app window when Edge is available, otherwise default browser.
		for _, p := range []string{filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe")} {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return exec.Command(p, "--app="+url, "--window-size=1360,850").Start()
			}
		}
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}
func (a *App) load() {
	a.config.Tags = make(map[string]string)
	a.config.Profiles = make(map[string][]string)
	data, err := os.ReadFile(a.configFile)
	if err == nil {
		if err := json.Unmarshal(data, &a.config); err != nil {
			log.Println("Config inválida:", err)
		}
	}
	if a.config.Tags == nil {
		a.config.Tags = make(map[string]string)
	}
	if a.config.Profiles == nil {
		a.config.Profiles = make(map[string][]string)
	}
	if a.config.PhotoFiles == nil {
		a.config.PhotoFiles = make(map[string]string)
	}
	if a.config.ModDir == "" {
		a.config.ModDir = autoDetect()
	}
}
func autoDetect() string {
	var paths []string
	for _, key := range []string{"APPDATA", "LOCALAPPDATA"} {
		if base := os.Getenv(key); base != "" {
			paths = append(paths,
				filepath.Join(base, "XXMI Launcher", "ZZMI", "Mods"),
				filepath.Join(base, "XXMI Launcher", "ZZMI", "ZZMI", "Mods"),
				filepath.Join(base, "XXMI-Launcher", "ZZMI", "Mods"))
		}
	}
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return ""
}
func (a *App) save() error {
	b, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return err
	}
	p := a.configFile + ".tmp"
	if err = os.WriteFile(p, b, 0600); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(a.configFile)
	}
	return os.Rename(p, a.configFile)
}
func validDir(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("Selecione a pasta Mods do ZZMI primeiro")
	}
	abs, err := filepath.Abs(strings.Trim(strings.TrimSpace(path), "\""))
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("Pasta não encontrada: %w", err)
	}
	if !st.IsDir() {
		return "", errors.New("O caminho selecionado não é uma pasta")
	}
	if !strings.EqualFold(filepath.Base(abs), "Mods") {
		return "", errors.New("Selecione a pasta chamada Mods (não a pasta do jogo nem a pasta ZZMI)")
	}
	return abs, nil
}
func canonRel(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, s := range parts {
		parts[i] = prettyName(s)
	}
	return strings.Join(parts, "/")
}
func isDisabled(n string) bool { return strings.HasPrefix(strings.ToUpper(n), "DISABLED") }
func prettyName(n string) string {
	if !isDisabled(n) {
		return n
	}
	return strings.TrimLeft(n[len("DISABLED"):], " _-")
}

var characterNames = []string{"Severian", "Phoenix", "Roxy", "Claret", "Sigrid", "Remielle", "Norma", "Velina", "Nangong", "Sunna", "Aria", "Miyabi", "Rina", "Koleda", "Nicole", "Anby", "Ellen", "Zhu Yuan", "Qingyi", "Jane", "Seth", "Caesar", "Burnice", "Lucy", "Piper", "Grace", "Soldier 11", "Soldado 11", "Nekomata", "Corin", "Billy", "Lycaon", "Soukaku", "Ben", "Anton", "Lighter", "Yanagi", "Harumasa", "Astra", "Evelyn", "Pulchra", "Trigger", "Soldier 0", "Vivian", "Hugo", "Yixuan", "Ju Fufu", "Alice", "Yuzuha", "Seed", "Orphie", "Magus", "Lucia", "Yidhari", "Pyrois", "Promeia", "Cissia", "Dialyn", "Banyue", "Zhao", "Ye Shunguang", "Pan Yinhu", "Komano Manato", "Starlight Billy", "Belle", "Wise"}

func guessCharacter(name string) string {
	lower := strings.ToLower(strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(name))
	best := ""
	for _, ch := range characterNames {
		if strings.Contains(lower, strings.ToLower(ch)) && len(ch) > len(best) {
			best = ch
		}
	}
	if best != "" {
		return best
	}
	return "Sem categoria"
}
func readHashes(iniPath string, hashes map[string]bool) {
	st, err := os.Stat(iniPath)
	if err != nil || st.Size() > 2<<20 {
		return
	}
	b, err := os.ReadFile(iniPath)
	if err != nil {
		return
	}
	for _, match := range hashRE.FindAllSubmatch(b, -1) {
		hashes[strings.ToLower(string(match[1]))] = true
	}
}
func scanMods(root string, tags map[string]string) ([]*Mod, []Conflict, error) {
	mods := make([]*Mod, 0)
	if root == "" {
		return mods, []Conflict{}, nil
	}
	root, err := validDir(root)
	if err != nil {
		return mods, nil, err
	}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		} // One unreadable mod should not break the whole library.
		if !d.IsDir() {
			return nil
		}
		if p == root {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") || strings.EqualFold(name, "ShaderFixes") || strings.EqualFold(name, "ShaderCache") || strings.EqualFold(name, "BufferValues") {
			return filepath.SkipDir
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return filepath.SkipDir
		}
		var hasIni bool
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".ini") {
				hasIni = true
				break
			}
		}
		if !hasIni {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return filepath.SkipDir
		}
		id := canonRel(rel)
		parts := strings.Split(filepath.ToSlash(rel), "/")
		inherited := false
		for _, part := range parts[:len(parts)-1] {
			if isDisabled(part) {
				inherited = true
			}
		}
		enabled := !isDisabled(name) && !inherited
		mod := &Mod{ID: id, Name: prettyName(name), RelPath: filepath.ToSlash(rel), Enabled: enabled, InheritedDisabled: inherited, path: p, hashes: make(map[string]bool)}
		mod.Hotkeys = []ModHotkey{}
		mod.Character = guessCharacter(mod.Name)
		if t := tags[id]; t != "" {
			mod.Character = t
		}
		var bestPreview string
		var bestScore int
		_ = filepath.WalkDir(p, func(filePath string, entry fs.DirEntry, readErr error) error {
			if readErr != nil {
				return nil
			}
			if entry.IsDir() {
				if filePath != p && strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".ini" {
				mod.IniCount++
				mod.Hotkeys = readModHotkeys(filePath, p, mod.Hotkeys)
				if enabled {
					readHashes(filePath, mod.hashes)
				}
			}
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp" {
				low := strings.ToLower(entry.Name())
				score := 0
				for _, word := range []string{"preview", "thumb", "cover", "screenshot", "image", "photo", "portrait", "展示", "预览"} {
					if strings.Contains(low, word) {
						score += 5
					}
				}
				depth := strings.Count(strings.TrimPrefix(filePath, p), string(filepath.Separator))
				if depth <= 2 {
					score += 2
				}
				if strings.Contains(low, "texture") || strings.Contains(low, "diffuse") || strings.Contains(low, "normal") || strings.Contains(low, "lightmap") {
					score -= 10
				}
				if score > bestScore {
					if stat, e := entry.Info(); e == nil && stat.Size() <= 12<<20 {
						bestScore = score
						bestPreview = filePath
					}
				}
			}
			return nil
		})
		mod.Preview = bestPreview != ""
		mod.previewPath = bestPreview
		if st, err := os.Stat(p); err == nil {
			mod.LastModified = st.ModTime().Format("02/01/2006")
		}
		mods = append(mods, mod)
		return filepath.SkipDir
	})
	if err != nil {
		return mods, nil, err
	}
	sort.Slice(mods, func(i, j int) bool { return strings.ToLower(mods[i].Name) < strings.ToLower(mods[j].Name) })
	hashMods := make(map[string][]*Mod)
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		for h := range m.hashes {
			hashMods[h] = append(hashMods[h], m)
		}
	}
	conflicts := []Conflict{}
	for h, owners := range hashMods {
		if len(owners) < 2 {
			continue
		}
		c := Conflict{Hash: h, ModNames: []string{}, ModIDs: []string{}}
		for _, m := range owners {
			c.ModNames = append(c.ModNames, m.Name)
			c.ModIDs = append(c.ModIDs, m.ID)
			m.Conflicts++
		}
		conflicts = append(conflicts, c)
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Hash < conflicts[j].Hash })
	return mods, conflicts, nil
}
func (a *App) state() (State, error) {
	result := State{ModDir: a.config.ModDir, Mods: []*Mod{}, Profiles: []Profile{}, Conflicts: []Conflict{}, Favorites: append([]string{}, a.config.Favorites...), CustomPhotos: []string{}}
	for n := range a.config.PhotoFiles {
		result.CustomPhotos = append(result.CustomPhotos, n)
	}
	sort.Strings(result.CustomPhotos)
	for n, ids := range a.config.Profiles {
		result.Profiles = append(result.Profiles, Profile{Name: n, Count: len(ids)})
	}
	sort.Slice(result.Profiles, func(i, j int) bool {
		return strings.ToLower(result.Profiles[i].Name) < strings.ToLower(result.Profiles[j].Name)
	})
	if a.config.ModDir == "" {
		return result, nil
	}
	mods, conflicts, err := scanMods(a.config.ModDir, a.config.Tags)
	if err != nil {
		return result, err
	}
	result.Configured = true
	result.Mods = mods
	result.Conflicts = conflicts
	result.Installed = len(mods)
	for _, m := range mods {
		if m.Enabled {
			result.Active++
		} else {
			result.Inactive++
		}
	}
	a.previewCache = make(map[string]string)
	for _, m := range mods {
		if m.Preview {
			a.previewCache[m.ID] = m.previewPath
		}
	}
	return result, nil
}
func jsonWrite(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func apiFail(w http.ResponseWriter, status int, msg string) {
	jsonWrite(w, status, APIError{Error: msg})
}
func decodeRequest(r *http.Request, dst any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 128*1024)).Decode(dst)
}
func (a *App) api(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Desk-Token") != a.token && !(r.Method == "GET" && (r.URL.Path == "/api/preview" || r.URL.Path == "/api/agent-photo") && r.URL.Query().Get("token") == a.token) {
		apiFail(w, 403, "Acesso não autorizado")
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		apiFail(w, 405, "Método inválido")
		return
	}
	if a.updateAPI(w, r) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	name := strings.TrimPrefix(r.URL.Path, "/api/")
	switch name {
	case "state":
		if r.Method != "GET" {
			break
		}
		state, err := a.state()
		if err != nil {
			apiFail(w, 400, err.Error())
			return
		}
		jsonWrite(w, 200, state)
		return
	case "configure":
		if r.Method != "POST" {
			break
		}
		var req struct {
			Path string `json:"path"`
		}
		if err := decodeRequest(r, &req); err != nil {
			apiFail(w, 400, "Caminho inválido")
			return
		}
		p, err := validDir(req.Path)
		if err != nil {
			apiFail(w, 400, err.Error())
			return
		}
		a.config.ModDir = p
		if err := a.save(); err != nil {
			apiFail(w, 500, err.Error())
			return
		}
		jsonWrite(w, 200, map[string]string{"message": "Pasta configurada"})
		return
	case "pick-folder":
		if r.Method != "POST" {
			break
		}
		p, err := pickFolder()
		if err != nil {
			apiFail(w, 400, err.Error())
			return
		}
		jsonWrite(w, 200, map[string]string{"path": p})
		return
	case "toggle":
		if r.Method != "POST" {
			break
		}
		var req struct {
			ID      string `json:"id"`
			Enabled bool   `json:"enabled"`
		}
		if err := decodeRequest(r, &req); err != nil {
			apiFail(w, 400, "Dados inválidos")
			return
		}
		mods, _, err := scanMods(a.config.ModDir, a.config.Tags)
		if err != nil {
			apiFail(w, 400, err.Error())
			return
		}
		m := findMod(mods, req.ID)
		if m == nil {
			apiFail(w, 404, "Mod não encontrado")
			return
		}
		if err := toggleMod(m, req.Enabled); err != nil {
			apiFail(w, 400, err.Error())
			return
		}
		jsonWrite(w, 200, map[string]string{"message": "Estado atualizado"})
		return
	case "tag":
		if r.Method != "POST" {
			break
		}
		var req struct {
			ID        string `json:"id"`
			Character string `json:"character"`
		}
		if err := decodeRequest(r, &req); err != nil {
			apiFail(w, 400, "Dados inválidos")
			return
		}
		mods, _, err := scanMods(a.config.ModDir, a.config.Tags)
		if err != nil {
			apiFail(w, 400, err.Error())
			return
		}
		if findMod(mods, req.ID) == nil {
			apiFail(w, 404, "Mod não encontrado")
			return
		}
		req.Character = strings.TrimSpace(req.Character)
		if len([]rune(req.Character)) > 60 {
			apiFail(w, 400, "Categoria muito longa")
			return
		}
		if req.Character == "" {
			delete(a.config.Tags, req.ID)
		} else {
			a.config.Tags[req.ID] = req.Character
		}
		if err := a.save(); err != nil {
			apiFail(w, 500, err.Error())
			return
		}
		jsonWrite(w, 200, map[string]string{"message": "Categoria salva"})
		return
	case "save-profile":
		if r.Method != "POST" {
			break
		}
		var req struct {
			Name string    `json:"name"`
			IDs  *[]string `json:"ids"`
		}
		if err := decodeRequest(r, &req); err != nil {
			apiFail(w, 400, "Nome inválido")
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" || len([]rune(req.Name)) > 48 {
			apiFail(w, 400, "Use um nome de até 48 caracteres")
			return
		}
		mods, _, err := scanMods(a.config.ModDir, a.config.Tags)
		if err != nil {
			apiFail(w, 400, err.Error())
			return
		}
		ids := []string{}
		if req.IDs != nil {
			seen := map[string]bool{}
			for _, id := range *req.IDs {
				id = strings.TrimSpace(id)
				if id == "" || seen[id] {
					continue
				}
				if findMod(mods, id) == nil {
					apiFail(w, 404, "Mod não encontrado: "+id)
					return
				}
				seen[id] = true
				ids = append(ids, id)
			}
		} else {
			for _, m := range mods {
				if m.Enabled {
					ids = append(ids, m.ID)
				}
			}
		}
		a.config.Profiles[req.Name] = ids
		if err := a.save(); err != nil {
			apiFail(w, 500, err.Error())
			return
		}
		jsonWrite(w, 200, map[string]string{"message": "Perfil salvo"})
		return
	case "apply-profile":
		if r.Method != "POST" {
			break
		}
		var req struct {
			Name  string `json:"name"`
			Scope string `json:"scope"`
		}
		if err := decodeRequest(r, &req); err != nil {
			apiFail(w, 400, "Nome inválido")
			return
		}
		ids, ok := a.config.Profiles[req.Name]
		if !ok {
			apiFail(w, 404, "Perfil não encontrado")
			return
		}
		mods, _, err := scanMods(a.config.ModDir, a.config.Tags)
		if err != nil {
			apiFail(w, 400, err.Error())
			return
		}
		if req.Scope != "" && req.Scope != req.Name {
			apiFail(w, 400, "Escopo de agente inválido")
			return
		}
		selected := map[string]bool{}
		for _, id := range ids {
			selected[id] = true
		}
		// Do not partly apply profiles if a saved mod is missing or inherited-disabled.
		for _, id := range ids {
			m := findMod(mods, id)
			if m == nil {
				apiFail(w, 409, "Perfil contém mod não encontrado: "+id)
				return
			}
			if m.InheritedDisabled {
				apiFail(w, 409, "Há pasta superior desativada: "+id)
				return
			}
		}
		type change struct {
			m  *Mod
			to bool
		}
		changes := []change{}
		for _, m := range mods {
			if req.Scope != "" && !strings.EqualFold(m.Character, req.Scope) {
				continue
			}
			if m.Enabled && !selected[m.ID] {
				changes = append(changes, change{m, false})
			}
		}
		for _, m := range mods {
			if req.Scope != "" && !strings.EqualFold(m.Character, req.Scope) {
				continue
			}
			if !m.Enabled && selected[m.ID] {
				changes = append(changes, change{m, true})
			}
		}
		var completed []change
		for _, c := range changes {
			if err := toggleMod(c.m, c.to); err != nil {
				for i := len(completed) - 1; i >= 0; i-- {
					_ = toggleMod(completed[i].m, !completed[i].to)
				}
				apiFail(w, 400, "Erro ao aplicar perfil (alterações anteriores revertidas): "+err.Error())
				return
			}
			completed = append(completed, c)
		}
		jsonWrite(w, 200, map[string]string{"message": "Perfil aplicado"})
		return
	case "delete-profile":
		if r.Method != "POST" {
			break
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := decodeRequest(r, &req); err != nil {
			apiFail(w, 400, "Nome inválido")
			return
		}
		delete(a.config.Profiles, req.Name)
		if err := a.save(); err != nil {
			apiFail(w, 500, err.Error())
			return
		}
		jsonWrite(w, 200, map[string]string{"message": "Perfil excluído (mods preservados)"})
		return
	case "import":
		if r.Method != "POST" {
			break
		}
		modName, err := importArchive(a.config.ModDir, r.URL.Query().Get("name"), io.LimitReader(r.Body, maxUpload+1))
		if err != nil {
			apiFail(w, 400, err.Error())
			return
		}
		jsonWrite(w, 200, map[string]string{"message": "Pacote importado: " + modName})
		return
	case "import-folder":
		if r.Method != "POST" { break }
		parts, err := r.MultipartReader()
		if err != nil { apiFail(w, 400, "Pasta inválida"); return }
		modName, err := importFolder(a.config.ModDir, r.URL.Query().Get("name"), parts)
		if err != nil { apiFail(w, 400, err.Error()); return }
		jsonWrite(w, 200, map[string]string{"message": "Pasta importada: " + modName})
		return
	case "agent-photo":
		if r.Method != "GET" {
			break
		}
		name := r.URL.Query().Get("name")
		file := a.config.PhotoFiles[name]
		if file == "" {
			http.NotFound(w, r)
			return
		}
		if filepath.Base(file) != file {
			http.NotFound(w, r)
			return
		}
		path := filepath.Join(filepath.Dir(a.configFile), "photos", file)
		data, err := os.ReadFile(path)
		if err != nil || len(data) > 12<<20 {
			http.NotFound(w, r)
			return
		}
		typ := http.DetectContentType(data)
		if typ != "image/png" && typ != "image/jpeg" && typ != "image/webp" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", typ)
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(data)
		return
	case "set-agent-photo":
		if r.Method != "POST" {
			break
		}
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name == "" || len([]rune(name)) > 64 {
			apiFail(w, 400, "Agente inválido")
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20+1))
		if err != nil || len(body) > 8<<20 || len(body) < 8 {
			apiFail(w, 400, "Imagem inválida ou maior que 8 MB")
			return
		}
		typ := http.DetectContentType(body)
		ext := ""
		switch typ {
		case "image/png":
			ext = ".png"
		case "image/jpeg":
			ext = ".jpg"
		case "image/webp":
			ext = ".webp"
		default:
			apiFail(w, 400, "Use PNG, JPG ou WEBP")
			return
		}
		photoDir := filepath.Join(filepath.Dir(a.configFile), "photos")
		if err := os.MkdirAll(photoDir, 0700); err != nil {
			apiFail(w, 500, "Não foi possível criar pasta de imagens")
			return
		}
		hash := sha256.Sum256([]byte(name))
		filename := hex.EncodeToString(hash[:16]) + ext
		if err := os.WriteFile(filepath.Join(photoDir, filename), body, 0600); err != nil {
			apiFail(w, 500, "Falha ao salvar foto")
			return
		}
		old := a.config.PhotoFiles[name]
		a.config.PhotoFiles[name] = filename
		if err := a.save(); err != nil {
			apiFail(w, 500, err.Error())
			return
		}
		if old != "" && old != filename {
			_ = os.Remove(filepath.Join(photoDir, old))
		}
		jsonWrite(w, 200, map[string]string{"message": "Foto personalizada salva"})
		return
	case "delete-agent-photo":
		if r.Method != "POST" {
			break
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := decodeRequest(r, &req); err != nil {
			apiFail(w, 400, "Dados inválidos")
			return
		}
		old := a.config.PhotoFiles[req.Name]
		delete(a.config.PhotoFiles, req.Name)
		if err := a.save(); err != nil {
			apiFail(w, 500, err.Error())
			return
		}
		if old != "" && filepath.Base(old) == old {
			_ = os.Remove(filepath.Join(filepath.Dir(a.configFile), "photos", old))
		}
		jsonWrite(w, 200, map[string]string{"message": "Foto restaurada"})
		return
	case "favorite":
		if r.Method != "POST" {
			break
		}
		var req struct {
			Name     string `json:"name"`
			Favorite bool   `json:"favorite"`
		}
		if err := decodeRequest(r, &req); err != nil || len([]rune(req.Name)) > 64 || strings.TrimSpace(req.Name) == "" {
			apiFail(w, 400, "Agente inválido")
			return
		}
		all := make([]string, 0, len(a.config.Favorites)+1)
		for _, name := range a.config.Favorites {
			if name != req.Name {
				all = append(all, name)
			}
		}
		if req.Favorite {
			all = append(all, req.Name)
		}
		a.config.Favorites = all
		if err := a.save(); err != nil {
			apiFail(w, 500, err.Error())
			return
		}
		jsonWrite(w, 200, map[string]string{"message": "Favoritos atualizados"})
		return
	case "preview":
		if r.Method != "GET" {
			break
		}
		if a.previewCache == nil {
			_, _ = a.state()
		}
		p := a.previewCache[r.URL.Query().Get("id")]
		if p == "" {
			http.NotFound(w, r)
			return
		}
		root, err := validDir(a.config.ModDir)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		rel, err := filepath.Rel(root, p)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			http.NotFound(w, r)
			return
		}
		st, err := os.Stat(p)
		if err != nil || st.IsDir() || st.Size() > 12<<20 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, p)
		return
	case "open-folder":
		if r.Method != "POST" {
			break
		}
		var req struct {
			ID string `json:"id"`
		}
		if err := decodeRequest(r, &req); err != nil {
			apiFail(w, 400, "Dados inválidos")
			return
		}
		path := a.config.ModDir
		if req.ID != "" {
			mods, _, err := scanMods(a.config.ModDir, a.config.Tags)
			if err != nil {
				apiFail(w, 400, err.Error())
				return
			}
			m := findMod(mods, req.ID)
			if m == nil {
				apiFail(w, 404, "Mod não encontrado")
				return
			}
			path = m.path
		}
		if path == "" {
			apiFail(w, 400, "Configure a pasta primeiro")
			return
		}
		if err := openFolder(path); err != nil {
			apiFail(w, 500, "Não foi possível abrir a pasta: "+err.Error())
			return
		}
		jsonWrite(w, 200, map[string]string{"message": "Pasta aberta"})
		return
	case "quit":
		if r.Method != "POST" {
			break
		}
		jsonWrite(w, 200, map[string]string{"message": "Aplicativo encerrado"})
		go func() { time.Sleep(250 * time.Millisecond); os.Exit(0) }()
		return
	}
	apiFail(w, 404, "Rota não encontrada")
}
func findMod(mods []*Mod, id string) *Mod {
	for _, m := range mods {
		if m.ID == id {
			return m
		}
	}
	return nil
}
func toggleMod(m *Mod, enable bool) error {
	if m.InheritedDisabled && enable {
		return errors.New("Uma pasta acima desse mod está desativada; ative-a manualmente")
	}
	current := filepath.Base(m.path)
	if isDisabled(current) == !enable {
		return nil
	}
	newName := current
	if enable {
		newName = prettyName(current)
	} else {
		newName = disabledPrefix + current
	}
	if newName == "" {
		return errors.New("Nome de pasta inválido")
	}
	target := filepath.Join(filepath.Dir(m.path), newName)
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("Já existe uma pasta com o nome %s", newName)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(m.path, target); err != nil {
		return fmt.Errorf("Não foi possível renomear o mod: %w", err)
	}
	m.path = target
	m.Enabled = enable
	return nil
}
func pickFolder() (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("Escolha a pasta copiando o caminho nas configurações")
	}
	ps := "Add-Type -AssemblyName System.Windows.Forms; $f=New-Object System.Windows.Forms.FolderBrowserDialog; $f.Description='Selecione a pasta Mods do ZZMI'; if($f.ShowDialog() -eq 'OK'){[Console]::Write($f.SelectedPath)}"
	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-Command", ps)
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Seleção cancelada ou indisponível: %w", err)
	}
	p := strings.TrimSpace(string(b))
	if p == "" {
		return "", errors.New("Seleção cancelada")
	}
	return p, nil
}
func openFolder(path string) error {
	if runtime.GOOS == "windows" {
		return exec.Command("explorer.exe", path).Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", path).Start()
	}
	return exec.Command("xdg-open", path).Start()
}
func zipSegmentOK(part string) bool {
	if part == "" || part == "." || part == ".." || len(part) > 180 || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") || invalidWindowsNameRE.MatchString(part) {
		return false
	}
	upper := strings.ToUpper(strings.Split(part, ".")[0])
	switch upper {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	}
	return true
}
func cleanArchiveName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = invalidWindowsNameRE.ReplaceAllString(name, "_")
	name = strings.Trim(name, ". ")
	if len(name) > 80 {
		name = name[:80]
	}
	if !zipSegmentOK(name) || name == "" {
		name = "Novo mod"
	}
	return name
}
func importZIP(root, name string, body io.Reader) (string, error) {
	root, err := validDir(root)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(name), ".zip") {
		return "", errors.New("Use arquivos .zip; por segurança não executo instaladores nem arquivos .exe")
	}
	tmp, err := os.CreateTemp("", "zzz-mod-upload-*.zip")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	n, err := io.Copy(tmp, body)
	if err != nil {
		return "", err
	}
	if n > maxUpload {
		return "", errors.New("ZIP maior que 1 GB")
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	archive, err := zip.OpenReader(tmp.Name())
	if err != nil {
		return "", errors.New("O ZIP está corrompido ou é inválido")
	}
	defer archive.Close()
	if len(archive.File) > 8000 {
		return "", errors.New("ZIP contém arquivos demais")
	}
	var total int64
	var foundINI bool
	for _, f := range archive.File {
		path := strings.ReplaceAll(f.Name, "\\", "/")
		path = strings.TrimSuffix(path, "/")
		if path == "" {
			continue
		}
		for _, part := range strings.Split(path, "/") {
			if !zipSegmentOK(part) {
				return "", fmt.Errorf("Caminho inseguro no ZIP: %s", f.Name)
			}
		}
		// Mod archives should not install executables, shortcuts, or scripts.
		switch strings.ToLower(filepath.Ext(path)) {
		case ".exe", ".msi", ".cmd", ".bat", ".com", ".scr", ".ps1", ".vbs", ".vbe", ".hta", ".lnk", ".url", ".reg", ".pif":
			return "", fmt.Errorf("ZIP contém arquivo executável ou atalho não permitido: %s", f.Name)
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("ZIP contém link simbólico; importação bloqueada")
		}
		if f.Mode()&os.ModeType != 0 && !f.FileInfo().IsDir() {
			return "", errors.New("ZIP contém arquivo especial; importação bloqueada")
		}
		if f.UncompressedSize64 > uint64(maxExtract) {
			return "", errors.New("Um arquivo é grande demais")
		}
		total += int64(f.UncompressedSize64)
		if total > maxExtract {
			return "", errors.New("ZIP descompactado ultrapassaria 2 GB")
		}
		if strings.EqualFold(filepath.Ext(path), ".ini") {
			foundINI = true
		}
	}
	if !foundINI {
		return "", errors.New("Nenhum arquivo .ini de mod encontrado neste ZIP")
	}
	base := cleanArchiveName(name)
	var dest string
	for i := 1; i < 10000; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s (%d)", base, i)
		}
		dest = filepath.Join(root, candidate)
		if _, err := os.Lstat(dest); os.IsNotExist(err) {
			break
		} else if err != nil {
			return "", err
		}
		dest = ""
	}
	if dest == "" {
		return "", errors.New("Não foi possível encontrar um nome disponível")
	}
	if err := os.Mkdir(dest, 0755); err != nil {
		return "", err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.RemoveAll(dest)
		}
	}()
	var extracted int64
	for _, f := range archive.File {
		rel := strings.TrimSuffix(strings.ReplaceAll(f.Name, "\\", "/"), "/")
		if rel == "" {
			continue
		}
		out := filepath.Join(dest, filepath.FromSlash(rel))
		r, err := filepath.Rel(dest, out)
		if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			return "", errors.New("Arquivo fora do diretório de destino")
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(out, 0755); err != nil {
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
			return "", err
		}
		reader, err := f.Open()
		if err != nil {
			return "", err
		}
		writer, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			reader.Close()
			return "", err
		}
		count, err := io.Copy(writer, io.LimitReader(reader, maxExtract-extracted+1))
		closeErr := writer.Close()
		reader.Close()
		extracted += count
		if err != nil {
			return "", err
		}
		if closeErr != nil {
			return "", closeErr
		}
		if extracted > maxExtract {
			return "", errors.New("Arquivos descompactados excedem 2 GB")
		}
		if count != int64(f.UncompressedSize64) {
			return "", errors.New("Tamanho inesperado em arquivo do ZIP")
		}
	}
	ok = true
	return filepath.Base(dest), nil
}