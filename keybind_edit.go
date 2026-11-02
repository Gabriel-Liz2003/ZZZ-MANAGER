package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

type hotkeyChange struct {
	ID         string `json:"id"`
	File       string `json:"file"`
	Section    string `json:"section"`
	Occurrence int    `json:"occurrence"`
	Field      string `json:"field"`
	Index      int    `json:"index"`
	Previous   string `json:"previous"`
	Key        string `json:"key"`
}

var functionHotkeyRE = regexp.MustCompile(`^F(?:[1-9]|1[0-9]|2[0-4])$`)

func validNewHotkey(value string) bool {
	if value == "" || len(value) > 80 || strings.ContainsAny(value, "\r\n;#=") {
		return false
	}
	parts := strings.Fields(value)
	if len(parts) < 1 || len(parts) > 4 {
		return false
	}
	modifiers := make(map[string]bool)
	for _, part := range parts[:len(parts)-1] {
		part = strings.ToLower(part)
		if part != "ctrl" && part != "alt" && part != "shift" || modifiers[part] {
			return false
		}
		modifiers[part] = true
	}
	key := strings.ToUpper(parts[len(parts)-1])
	if len(key) == 1 && (key[0] >= 'A' && key[0] <= 'Z' || key[0] >= '0' && key[0] <= '9') {
		return true
	}
	if functionHotkeyRE.MatchString(key) {
		return true
	}
	if strings.HasPrefix(key, "VK_NUMPAD") {
		number := strings.TrimPrefix(key, "VK_NUMPAD")
		n, err := strconv.Atoi(number)
		return err == nil && len(number) == 1 && n >= 0 && n <= 9
	}
	switch key {
	case "VK_LEFT", "VK_RIGHT", "VK_UP", "VK_DOWN", "VK_SPACE", "VK_RETURN",
		"VK_BACK", "VK_TAB", "VK_DELETE", "VK_INSERT", "VK_HOME", "VK_END",
		"VK_PRIOR", "VK_NEXT", "VK_ADD", "VK_SUBTRACT", "VK_MULTIPLY",
		"VK_DIVIDE", "VK_DECIMAL", "VK_OEM_PLUS", "VK_OEM_MINUS",
		"VK_OEM_1", "VK_OEM_2", "VK_OEM_3", "VK_OEM_4", "VK_OEM_5",
		"VK_OEM_6", "VK_OEM_7", "VK_OEM_COMMA", "VK_OEM_PERIOD",
		"VK_ESCAPE", "VK_CAPITAL", "VK_PAUSE":
		return true
	}
	return false
}

func decodedHotkeyINI(data []byte) (string, int, error) {
	if len(data) >= 2 && (bytes.Equal(data[:2], []byte{0xff, 0xfe}) || bytes.Equal(data[:2], []byte{0xfe, 0xff})) {
		if (len(data)-2)%2 != 0 {
			return "", 0, errors.New("Arquivo UTF-16 incompleto")
		}
		little := data[0] == 0xff
		codeUnits := make([]uint16, 0, (len(data)-2)/2)
		for i := 2; i < len(data); i += 2 {
			if little {
				codeUnits = append(codeUnits, binary.LittleEndian.Uint16(data[i:i+2]))
			} else {
				codeUnits = append(codeUnits, binary.BigEndian.Uint16(data[i:i+2]))
			}
		}
		if little {
			return string(utf16.Decode(codeUnits)), 2, nil
		}
		return string(utf16.Decode(codeUnits)), 3, nil
	}
	if bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return string(data[3:]), 1, nil
	}
	// Keep raw legacy ANSI bytes intact instead of replacing unknown characters.
	return string(data), 0, nil
}

func encodedHotkeyINI(text string, encoding int) []byte {
	switch encoding {
	case 1:
		return append([]byte{0xef, 0xbb, 0xbf}, []byte(text)...)
	case 2, 3:
		units := utf16.Encode([]rune(text))
		out := make([]byte, 2+len(units)*2)
		if encoding == 2 {
			copy(out[:2], []byte{0xff, 0xfe})
		} else {
			copy(out[:2], []byte{0xfe, 0xff})
		}
		for i, code := range units {
			if encoding == 2 {
				binary.LittleEndian.PutUint16(out[2+i*2:], code)
			} else {
				binary.BigEndian.PutUint16(out[2+i*2:], code)
			}
		}
		return out
	default:
		return []byte(text)
	}
}

// replaceHotkeyINI changes one specific key/back assignment, not other
// declarations with the same field name in the file or in other sections.
func replaceHotkeyINI(data []byte, req hotkeyChange) ([]byte, error) {
	text, encoding, err := decodedHotkeyINI(data)
	if err != nil {
		return nil, err
	}
	lines := strings.SplitAfter(text, "\n")
	occurrences := map[string]int{}
	current := false
	fieldNumber := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			current = false
			if matches := hotkeyHeaderRE.FindStringSubmatch(trimmed); len(matches) == 2 {
				name := strings.TrimSpace(matches[1])
				if (strings.EqualFold(name, "Key") || isInterestingHotkey(name)) && !strings.HasPrefix(strings.ToLower(name), "keyboard") {
					identifier := strings.ToLower(name)
					occurrence := occurrences[identifier]
					occurrences[identifier]++
					current = strings.EqualFold(name, req.Section) && occurrence == req.Occurrence
					fieldNumber = 0
				}
			}
			continue
		}
		if !current {
			continue
		}
		bare := strings.TrimRight(line, "\r\n")
		ending := line[len(bare):]
		eq := strings.IndexByte(bare, '=')
		if eq < 1 || !strings.EqualFold(strings.TrimSpace(bare[:eq]), req.Field) {
			continue
		}
		old := stripINIComment(bare[eq+1:])
		if old == "" {
			continue
		}
		if fieldNumber != req.Index {
			fieldNumber++
			continue
		}
		if old != req.Previous {
			return nil, errors.New("O atalho mudou no arquivo desde a última leitura; atualize a biblioteca")
		}
		afterEq := bare[eq+1:]
		commentPos := strings.IndexAny(afterEq, ";#")
		valuePart, comment := afterEq, ""
		if commentPos >= 0 {
			valuePart, comment = afterEq[:commentPos], afterEq[commentPos:]
		}
		left := len(valuePart) - len(strings.TrimLeft(valuePart, " \t"))
		right := len(valuePart) - len(strings.TrimRight(valuePart, " \t"))
		lines[i] = bare[:eq+1] + valuePart[:left] + req.Key + valuePart[len(valuePart)-right:] + comment + ending
		return encodedHotkeyINI(strings.Join(lines, ""), encoding), nil
	}
	return nil, errors.New("Atalho não encontrado no arquivo .ini")
}

// editModHotkey only accepts shortcuts already discovered within this mod.
// It never uses an arbitrary path received from the browser for a file write.
func editModHotkey(mod *Mod, req hotkeyChange) error {
	if req.Field != "key" && req.Field != "back" {
		return errors.New("Campo de atalho inválido")
	}
	if req.Index < 0 || req.Occurrence < 0 || !validNewHotkey(req.Key) {
		return errors.New("Tecla inválida ou não suportada")
	}
	var keys []string
	allowed := false
	for _, item := range mod.Hotkeys {
		if item.File == req.File && item.Section == req.Section && item.Occurrence == req.Occurrence {
			keys = item.Keys
			if req.Field == "back" {
				keys = item.Back
			}
			allowed = true
			break
		}
	}
	if !allowed || req.Index >= len(keys) {
		return errors.New("Comando não encontrado neste mod")
	}
	if keys[req.Index] != req.Previous {
		return errors.New("O atalho foi alterado; atualize a biblioteca antes de tentar novamente")
	}
	file := filepath.Join(mod.path, filepath.FromSlash(req.File))
	rel, err := filepath.Rel(mod.path, file)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("Caminho do .ini inválido")
	}
	info, err := os.Lstat(file)
	if err != nil {
		return fmt.Errorf("Não foi possível acessar o .ini: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > maxHotkeyINIBytes || !strings.EqualFold(filepath.Ext(file), ".ini") {
		return errors.New("O arquivo precisa ser um .ini comum de até 2 MB, sem links simbólicos")
	}
	before, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	after, err := replaceHotkeyINI(before, req)
	if err != nil {
		return err
	}
	if bytes.Equal(before, after) {
		return nil
	}
	backup := file + ".zzzmanager.bak"
	f, err := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err == nil {
		if _, err = f.Write(before); err != nil {
			_ = f.Close()
			_ = os.Remove(backup)
			return fmt.Errorf("Falha ao criar backup: %w", err)
		}
		err = f.Close()
	} else if errors.Is(err, os.ErrExist) {
		err = nil // Preserve the original backup across future hotkey edits.
	} else {
		return fmt.Errorf("Falha ao criar backup: %w", err)
	}
	if err != nil {
		return err
	}
	if err := os.WriteFile(file, after, info.Mode().Perm()); err != nil {
		return fmt.Errorf("Não foi possível gravar o .ini: %w", err)
	}
	return nil
}
