package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// DiscordCatalogMod is a user-curated bookmark for a Discord attachment.
// The desktop manager does not automate Discord user accounts, read forum
// messages, or claim access to posts behind server permissions.
type DiscordCatalogMod struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Agent        string `json:"agent"`
	DownloadURL  string `json:"downloadUrl"`
	MessageURL   string `json:"messageUrl,omitempty"`
	PreviewURL   string `json:"previewUrl,omitempty"`
	InstallCount int    `json:"installCount"`
}

func isDiscordCDNURL(raw string, isArchive bool) (string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) == 0 || len(raw) > 4096 {
		return "", errors.New("Cole o link direto do anexo no Discord")
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil || !strings.EqualFold(u.Scheme, "https") || u.User != nil || u.Port() != "" || u.Fragment != "" {
		return "", errors.New("O link precisa ser HTTPS e apontar para um anexo do Discord")
	}
	switch strings.ToLower(u.Hostname()) {
	case "cdn.discordapp.com", "cdn.discord.com", "media.discordapp.net":
	default:
		return "", errors.New("Use o link direto do arquivo no CDN do Discord, não o link da mensagem")
	}
	if !strings.HasPrefix(u.EscapedPath(), "/attachments/") &&
		!strings.HasPrefix(u.EscapedPath(), "/ephemeral-attachments/") {
		return "", errors.New("O URL não aponta para um anexo do Discord")
	}
	if strings.Contains(u.Path, "..") || strings.ContainsAny(u.Path, "\r\n") {
		return "", errors.New("Caminho do anexo inválido")
	}
	ext := strings.ToLower(path.Ext(u.Path))
	if isArchive {
		switch ext {
		case ".zip", ".rar", ".7z":
		default:
			return "", errors.New("Selecione o link direto de um arquivo ZIP, RAR ou 7z")
		}
	} else {
		switch ext {
		case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		default:
			return "", errors.New("A prévia precisa ser uma imagem PNG, JPEG, WEBP ou GIF")
		}
	}
	return raw, nil
}

func validDiscordMessageURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if len(raw) > 2048 {
		return "", errors.New("O link da mensagem é muito longo")
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return "", errors.New("Link de mensagem inválido")
	}
	if u.Hostname() != "discord.com" && u.Hostname() != "discordapp.com" && u.Hostname() != "ptb.discord.com" && u.Hostname() != "canary.discord.com" {
		return "", errors.New("O link da publicação precisa ser do Discord")
	}
	if !strings.HasPrefix(u.Path, "/channels/") {
		return "", errors.New("Copie o link de uma mensagem ou publicação do Discord")
	}
	return raw, nil
}

func normalizeDiscordCatalogMod(item DiscordCatalogMod) (DiscordCatalogMod, error) {
	item.Name = strings.TrimSpace(item.Name)
	item.Agent = strings.TrimSpace(item.Agent)
	if len([]rune(item.Name)) < 2 || len([]rune(item.Name)) > 120 || strings.ContainsAny(item.Name, "\x00\r\n") {
		return item, errors.New("Escolha um nome de mod válido (2 a 120 caracteres)")
	}
	if !cleanAgentFolder(item.Agent) {
		return item, errors.New("Selecione uma agente válida")
	}
	var err error
	if item.DownloadURL, err = isDiscordCDNURL(item.DownloadURL, true); err != nil {
		return item, err
	}
	if item.MessageURL, err = validDiscordMessageURL(item.MessageURL); err != nil {
		return item, err
	}
	if item.PreviewURL != "" {
		if item.PreviewURL, err = isDiscordCDNURL(item.PreviewURL, false); err != nil {
			return item, err
		}
	}
	return item, nil
}

func newDiscordCatalogID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func findDiscordCatalogMod(items []DiscordCatalogMod, id string) (int, bool) {
	for i := range items {
		if items[i].ID == id {
			return i, true
		}
	}
	return -1, false
}

// downloadDiscordArchive reuses the existing archive importer and its
// extraction limits, executable blocking, collision avoidance and rollback.
func downloadDiscordArchive(ctx context.Context, root, directURL string, client *http.Client) (string, error) {
	raw, err := isDiscordCDNURL(directURL, true)
	if err != nil {
		return "", err
	}
	u, _ := url.Parse(raw)
	filename := path.Base(u.Path)
	if filename == "." || filename == "/" {
		return "", errors.New("Nome do anexo inválido")
	}
	ctx, cancel := context.WithTimeout(ctx, 7*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ZZZ-Mod-Desk/Discord-Catalog")
	if client == nil {
		client = &http.Client{
			Timeout: 7 * time.Minute,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return errors.New("Redirecionamentos demais no download")
				}
				_, err := isDiscordCDNURL(req.URL.String(), true)
				return err
			},
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Não foi possível baixar o anexo do Discord: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
		return "", errors.New("O anexo está indisponível ou o link expirou. Copie um novo link do arquivo no Discord")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("O Discord respondeu com HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxUpload {
		return "", errors.New("O arquivo ultrapassa 1 GB")
	}
	return importArchive(root, filename, io.LimitReader(resp.Body, maxUpload+1))
}

// tagNewDiscordArchive tags exactly the newly imported folder's mods,
// including nested packages, before invoking the existing agent organizer.
func (a *App) tagNewDiscordArchive(folder, agent string) (int, error) {
	if !cleanAgentFolder(agent) {
		return 0, errors.New("Agente inválida")
	}
	root := filepath.Join(a.config.ModDir, folder)
	mods, _, err := scanMods(a.config.ModDir, a.config.Tags)
	if err != nil {
		return 0, err
	}
	if a.config.Tags == nil {
		a.config.Tags = map[string]string{}
	}
	count := 0
	for _, m := range mods {
		rel, err := filepath.Rel(root, m.path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		a.config.Tags[m.ID] = agent
		count++
	}
	if count == 0 {
		return 0, errors.New("O arquivo foi baixado, mas não foi possível identificar o mod instalado")
	}
	if err := a.save(); err != nil {
		return count, err
	}
	_, err = a.organizeAgentFolders(root)
	return count, err
}

