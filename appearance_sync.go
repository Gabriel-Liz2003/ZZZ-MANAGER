package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Only replace literal numeric defaults explicitly declared as global persist.
// The other INI settings, keybinds and mod resources remain untouched.
var editablePersistentConstant = regexp.MustCompile(`(?i)^(\s*global\s+persist\s+\$)([a-z_][a-z0-9_]*)(\s*=\s*)([-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][-+]?[0-9]+)?)(\s*(?:[;#].*)?)$`)
var runtimeNumericValue = regexp.MustCompile(`^[-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][-+]?[0-9]+)?$`)

type AppearanceAutoStatus struct {
	Enabled   bool   `json:"enabled"`
	Message   string `json:"message"`
	LastSaved string `json:"lastSaved,omitempty"`
	Saved     int    `json:"saved"`
}

func runtimeINIPath(modsRoot string) string {
	return filepath.Join(filepath.Dir(modsRoot), "d3dx_user.ini")
}

// Read the persistent values actually saved by ZZMI on F10. The runtime file
// may be UTF-8, UTF-8-BOM or UTF-16. No guesses from mod defaults are made.
func runtimePersistentValues(modsRoot string) (map[string]string, error) {
	file := runtimeINIPath(modsRoot)
	stat, err := os.Lstat(file)
	if err != nil {
		return nil, fmt.Errorf("d3dx_user.ini não encontrado ao lado da pasta Mods: %w", err)
	}
	if !stat.Mode().IsRegular() || stat.Size() > 4<<20 {
		return nil, errors.New("d3dx_user.ini não é um arquivo comum válido (limite 4 MB)")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	text, _, err := decodedHotkeyINI(data)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 1 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		if !strings.HasPrefix(key, "$\\mods\\") {
			continue
		}
		// The runtime key contains relative path + ini filename + variable.
		// Reject malformed and empty names rather than matching another mod.
		relative := strings.ReplaceAll(strings.TrimPrefix(key, "$\\mods\\"), "\\", "/")
		if !strings.Contains(relative, ".ini/") || strings.Contains(relative, "../") {
			continue
		}
		value := stripINIComment(line[eq+1:])
		if !runtimeNumericValue.MatchString(value) {
			continue
		}
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			continue
		}
		result[relative] = value
	}
	return result, scanner.Err()
}

type persistentINIEdits struct {
	path   string
	before []byte
	after  []byte
	mode   fs.FileMode
}

func planPersistentModEdits(modsRoot string, m *Mod, values map[string]string) (int, []persistentINIEdits, error) {
	if !m.Enabled || m.InheritedDisabled {
		return 0, nil, errors.New("Mod desativado: não é seguro substituir os valores")
	}
	matched := 0
	edits := []persistentINIEdits{}
	err := filepath.WalkDir(m.path, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != m.path && (strings.HasPrefix(d.Name(), ".") || nestedModPartDisabled(m.path, filepath.Join(path, "dummy.ini"))) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".ini") {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("INI não é arquivo comum: %s", filepath.Base(path))
		}
		if info.Size() > maxHotkeyINIBytes {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text, encoding, err := decodedHotkeyINI(data)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(modsRoot, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errors.New("Arquivo INI fora da biblioteca de mods")
		}
		namespace := strings.ToLower(filepath.ToSlash(rel)) + "/"
		lines := strings.SplitAfter(text, "\n")
		inConstants, changed := false, false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "[") {
				inConstants = strings.EqualFold(trimmed, "[Constants]")
				continue
			}
			if !inConstants || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
				continue
			}
			bare := strings.TrimRight(line, "\r\n")
			matches := editablePersistentConstant.FindStringSubmatch(bare)
			if matches == nil {
				continue
			}
			val, ok := values[namespace+strings.ToLower(matches[2])]
			if !ok {
				continue
			}
			matched++
			if val != matches[4] {
				ending := line[len(bare):]
				lines[i] = matches[1]+matches[2]+matches[3]+val+matches[5]+ending
				changed = true
			}
		}
		if changed {
			edits = append(edits, persistentINIEdits{
				path: path, before: data, after: encodedHotkeyINI(strings.Join(lines, ""), encoding),
				mode: info.Mode().Perm(),
			})
		}
		return nil
	})
	return matched, edits, err
}

func applyPersistentEdits(edits []persistentINIEdits) error {
	// First make original backups (never replace an earlier backup), then write.
	// If one write fails, restore the preceding files of this single mod.
	for _, e := range edits {
		backup := e.path + ".zzzmanager.bak"
		out, err := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("Falha ao criar backup de %s: %w", filepath.Base(e.path), err)
		}
		_, writeErr := out.Write(e.before)
		closeErr := out.Close()
		if writeErr != nil || closeErr != nil {
			_ = os.Remove(backup)
			return fmt.Errorf("Backup de %s incompleto", filepath.Base(e.path))
		}
	}
	for i, e := range edits {
		if err := os.WriteFile(e.path, e.after, e.mode); err != nil {
			for j := i-1; j >= 0; j-- {
				_ = os.WriteFile(edits[j].path, edits[j].before, edits[j].mode)
			}
			return fmt.Errorf("Não foi possível gravar %s: %w", filepath.Base(e.path), err)
		}
	}
	return nil
}

func syncModPersistentDefaultsWithValues(modsRoot string, m *Mod, values map[string]string) (int, int, error) {
	matched, edits, err := planPersistentModEdits(modsRoot, m, values)
	if err != nil {
		return 0, 0, err
	}
	if matched == 0 {
		return 0, 0, errors.New("Nenhuma variável global persist correspondente encontrada neste mod")
	}
	if err := applyPersistentEdits(edits); err != nil {
		return matched, 0, err
	}
	return matched, len(edits), nil
}

// Keep a safe explicit helper for old integrations and existing regression tests.
// The UI no longer requires manual calls.
func syncModPersistentDefaults(modsRoot string, m *Mod) (int, error) {
	if !m.Enabled || m.InheritedDisabled {
		return 0, errors.New("Ative a skin antes de salvar a variação")
	}
	values, err := runtimePersistentValues(modsRoot)
	if err != nil {
		return 0, err
	}
	matched, _, err := syncModPersistentDefaultsWithValues(modsRoot, m, values)
	return matched, err
}

// A runtime write is accepted only after two identical observations. This
// avoids parsing d3dx_user.ini while the game is still in the middle of F10.
type runtimeFileSignature struct {
	modTime time.Time
	size    int64
	ready   bool
}

func signatureForRuntimeFile(path string) runtimeFileSignature {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4<<20 {
		return runtimeFileSignature{}
	}
	return runtimeFileSignature{modTime: info.ModTime(), size: info.Size(), ready: true}
}

type appearanceWatcher struct {
	root    string
	applied runtimeFileSignature
	pending runtimeFileSignature
}

func (w *appearanceWatcher) next(root string, sig runtimeFileSignature) bool {
	if root != w.root {
		w.root, w.applied, w.pending = root, sig, runtimeFileSignature{}
		return false // Never replay stale saved values on app launch/configure.
	}
	if !sig.ready || sig == w.applied {
		w.pending = runtimeFileSignature{}
		return false
	}
	if sig != w.pending {
		w.pending = sig
		return false
	}
	w.applied, w.pending = sig, runtimeFileSignature{}
	return true
}

// Select by exact mod namespace, not by the text parser's detection count.
// Some mods use UTF-16 INIs and do not show up in the older quick counter.
func hasRuntimeKeysForMod(values map[string]string, modRelativePath string) bool {
	prefix := strings.ToLower(filepath.ToSlash(modRelativePath)) + "/"
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func (a *App) watchAppearancePresets(interval time.Duration) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var watcher appearanceWatcher
	// Baseline existing saved runtime values on launch, before waiting.
	a.mu.Lock()
	root := a.config.ModDir
	a.appearanceSync = AppearanceAutoStatus{Enabled: true, Message: "Automático: ajuste as variações no ZZZ e pressione F10."}
	a.mu.Unlock()
	watcher.next(root, signatureForRuntimeFile(runtimeINIPath(root)))
	for range ticker.C {
		a.mu.Lock()
		root = a.config.ModDir
		if !watcher.next(root, signatureForRuntimeFile(runtimeINIPath(root))) {
			a.mu.Unlock()
			continue
		}
		values, err := runtimePersistentValues(root)
		if err != nil {
			a.appearanceSync.Message = "Aguardando gravação válida do d3dx_user.ini após F10."
			log.Printf("auto preset: %v", err)
			a.mu.Unlock()
			continue
		}
		mods, _, err := scanMods(root, a.config.Tags)
		if err != nil {
			a.appearanceSync.Message = "Não foi possível ler a biblioteca para salvar presets."
			log.Printf("auto preset: %v", err)
			a.mu.Unlock()
			continue
		}
		updated, failed := 0, 0
		for _, mod := range mods {
			if !mod.Enabled || !hasRuntimeKeysForMod(values, mod.RelPath) {
				continue
			}
			_, changes, syncErr := syncModPersistentDefaultsWithValues(root, mod, values)
			if syncErr == nil {
				updated += changes
			} else if !strings.Contains(syncErr.Error(), "Nenhuma variável global persist") {
				failed++
				log.Printf("auto preset (%s): %v", mod.ID, syncErr)
			}
		}
		if updated > 0 {
			a.appearanceSync.LastSaved = time.Now().Format("02/01/2006 15:04:05")
			a.appearanceSync.Saved += updated
		}
		if failed > 0 {
			a.appearanceSync.Message = fmt.Sprintf("Salvamento parcial: %d arquivo(s) atualizados, %d erro(s). Verifique app.log.", updated, failed)
		} else if updated > 0 {
			a.appearanceSync.Message = fmt.Sprintf("Variações atualizadas automaticamente em %d arquivo(s).", updated)
		} else {
			a.appearanceSync.Message = "F10 detectado. Nenhum valor persistente precisou de alteração."
		}
		a.mu.Unlock()
	}
}

