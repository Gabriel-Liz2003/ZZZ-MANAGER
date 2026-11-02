package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	appVersion       = "3.6.2"
	githubRepository = "Gabriel-Liz2003/ZZZ-MANAGER"
	updateAssetName  = "ZZZ-Mod-Desk.exe"
	updateMaxSize    = 100 << 20 // 100 MiB
)

var semverPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)
var updateClient = &http.Client{Timeout: 35 * time.Second}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}
type githubRelease struct {
	Tag        string         `json:"tag_name"`
	HTMLURL    string         `json:"html_url"`
	Prerelease bool           `json:"prerelease"`
	Draft      bool           `json:"draft"`
	Assets     []releaseAsset `json:"assets"`
}
type UpdateInfo struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	Available   bool   `json:"available"`
	DownloadURL string `json:"downloadUrl,omitempty"`
	ReleaseURL  string `json:"releaseUrl,omitempty"`
	Message     string `json:"message"`
}

func compareVersion(a, b string) (int, error) {
	x, y := semverPattern.FindStringSubmatch(a), semverPattern.FindStringSubmatch(b)
	if x == nil || y == nil {
		return 0, errors.New("Versão inválida")
	}
	for i := 1; i <= 3; i++ {
		v, _ := strconv.Atoi(x[i])
		w, _ := strconv.Atoi(y[i])
		if v < w {
			return -1, nil
		}
		if v > w {
			return 1, nil
		}
	}
	return 0, nil
}
func safeGitHubURL(raw string) error {
	// Redirects are followed only to HTTPS; release assets may redirect to GitHub CDN.
	u, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil || u.URL.Scheme != "https" {
		return errors.New("URL de download inválida")
	}
	return nil
}
func getLatestRelease(ctx context.Context, cli *http.Client, apiURL string) (githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ZZZ-Mod-Desk/"+appVersion)
	res, err := cli.Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("Não foi possível acessar o GitHub: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return githubRelease{}, errors.New("Ainda não há uma versão publicada no GitHub Releases")
	}
	if res.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("GitHub respondeu HTTP %d", res.StatusCode)
	}
	var r githubRelease
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&r); err != nil {
		return githubRelease{}, err
	}
	if r.Draft || r.Prerelease {
		return githubRelease{}, errors.New("A última versão não é uma release estável")
	}
	if !semverPattern.MatchString(r.Tag) {
		return githubRelease{}, errors.New("Versão publicada fora do padrão vX.Y.Z")
	}
	return r, nil
}
func findAsset(r githubRelease, name string) (releaseAsset, error) {
	for _, a := range r.Assets {
		if a.Name == name && a.Size > 0 && a.Size <= updateMaxSize && safeGitHubURL(a.URL) == nil {
			return a, nil
		}
	}
	return releaseAsset{}, fmt.Errorf("Release não contém o arquivo %s", name)
}
func queryUpdate(ctx context.Context) (UpdateInfo, githubRelease, error) {
	info := UpdateInfo{Current: appVersion, Latest: appVersion, Message: "Você está na versão atual."}
	r, err := getLatestRelease(ctx, updateClient, "https://api.github.com/repos/"+githubRepository+"/releases/latest")
	if err != nil {
		return info, r, err
	}
	info.Latest = strings.TrimPrefix(r.Tag, "v")
	info.ReleaseURL = r.HTMLURL
	comp, err := compareVersion(appVersion, r.Tag)
	if err != nil {
		return info, r, err
	}
	if comp >= 0 {
		return info, r, nil
	}
	asset, err := findAsset(r, updateAssetName)
	if err != nil {
		return info, r, err
	}
	if _, err := findAsset(r, "SHA256SUMS.txt"); err != nil {
		return info, r, errors.New("Release sem comprovante SHA-256")
	}
	info.Available = true
	info.DownloadURL = asset.URL
	info.Message = "Nova versão disponível: " + info.Latest
	return info, r, nil
}
func fetchBounded(ctx context.Context, cli *http.Client, url string, max int64) ([]byte, error) {
	if err := safeGitHubURL(url); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ZZZ-Mod-Desk/"+appVersion)
	res, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d durante download", res.StatusCode)
	}
	if res.ContentLength > max {
		return nil, errors.New("Arquivo excede limite de tamanho")
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errors.New("Arquivo excede limite de tamanho")
	}
	return data, nil
}
func expectedHash(sumfile []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(sumfile), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == filename && len(fields[0]) == 64 {
			if _, err := hex.DecodeString(fields[0]); err == nil {
				return strings.ToLower(fields[0]), nil
			}
		}
	}
	return "", errors.New("Hash SHA-256 não encontrado para o executável")
}
func downloadAndVerify(ctx context.Context, cli *http.Client, r githubRelease) (string, error) {
	binary, err := findAsset(r, updateAssetName)
	if err != nil {
		return "", err
	}
	sums, err := findAsset(r, "SHA256SUMS.txt")
	if err != nil {
		return "", err
	}
	manifest, err := fetchBounded(ctx, cli, sums.URL, 64<<10)
	if err != nil {
		return "", err
	}
	expected, err := expectedHash(manifest, updateAssetName)
	if err != nil {
		return "", err
	}
	newBytes, err := fetchBounded(ctx, cli, binary.URL, updateMaxSize)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(newBytes)
	if hex.EncodeToString(h[:]) != expected {
		return "", errors.New("Integridade SHA-256 inválida: atualização cancelada")
	}
	if len(newBytes) < 2 || string(newBytes[:2]) != "MZ" {
		return "", errors.New("Arquivo baixado não é um executável Windows")
	}
	folder, err := os.MkdirTemp("", "neon-mods-update-")
	if err != nil {
		return "", err
	}
	dest := filepath.Join(folder, updateAssetName)
	if err := os.WriteFile(dest, newBytes, 0600); err != nil {
		_ = os.RemoveAll(folder)
		return "", err
	}
	return dest, nil
}
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func installUpdateWindows(newExecutable string) error {
	if runtime.GOOS != "windows" {
		return errors.New("Instalação automática disponível somente no Windows")
	}
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, err = filepath.Abs(current)
	if err != nil {
		return err
	}
	if strings.EqualFold(filepath.Base(current), "go-build.exe") {
		return errors.New("Instalação não suportada em modo de desenvolvimento")
	}
	scriptPath := filepath.Join(filepath.Dir(newExecutable), "install-update.ps1")
	backup := current + ".backup"
	// Never touch Mods, configuration or user files. Process exits after acknowledging the request.
	script := fmt.Sprintf(`$ErrorActionPreference='Stop'
$target=%s
$source=%s
$backup=%s
$oldPid=%d
for($i=0; $i -lt 150; $i++) { if(-not (Get-Process -Id $oldPid -ErrorAction SilentlyContinue)) { break }; Start-Sleep -Milliseconds 300 }
if(Get-Process -Id $oldPid -ErrorAction SilentlyContinue) { exit 2 }
try {
  Copy-Item -LiteralPath $target -Destination $backup -Force
  Copy-Item -LiteralPath $source -Destination $target -Force
  Start-Process -FilePath $target
} catch {
  if(Test-Path -LiteralPath $backup) { Copy-Item -LiteralPath $backup -Destination $target -Force }
  exit 1
}
`, psQuote(current), psQuote(newExecutable), psQuote(backup), os.Getpid())
	if err := os.WriteFile(scriptPath, []byte(script), 0600); err != nil {
		return err
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
func (a *App) updateAPI(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/api/update/check":
		if r.Method != "GET" {
			apiFail(w, 405, "Método inválido")
			return true
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		info, _, err := queryUpdate(ctx)
		if err != nil {
			info.Message = err.Error()
			jsonWrite(w, 200, info)
			return true
		}
		jsonWrite(w, 200, info)
		return true
	case "/api/update/install":
		if r.Method != "POST" {
			apiFail(w, 405, "Método inválido")
			return true
		}
		if runtime.GOOS != "windows" {
			apiFail(w, 400, "Disponível somente no Windows")
			return true
		}
		var request struct {
			Version string `json:"version"`
		}
		if err := decodeRequest(r, &request); err != nil {
			apiFail(w, 400, "Pedido inválido")
			return true
		}
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
		defer cancel()
		info, release, err := queryUpdate(ctx)
		if err != nil {
			apiFail(w, 502, err.Error())
			return true
		}
		if !info.Available || request.Version != info.Latest {
			apiFail(w, 409, "A versão disponível mudou; verifique novamente")
			return true
		}
		staged, err := downloadAndVerify(ctx, updateClient, release)
		if err != nil {
			apiFail(w, 502, err.Error())
			return true
		}
		if err := installUpdateWindows(staged); err != nil {
			apiFail(w, 500, err.Error())
			return true
		}
		jsonWrite(w, 200, map[string]string{"message": "Atualização validada; reiniciando o aplicativo."})
		go func() { time.Sleep(300 * time.Millisecond); os.Exit(0) }()
		return true
	}
	return false
}
