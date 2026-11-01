package main

import (
    "bufio"
    "bytes"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "unicode"
    "unicode/utf16"
    "unicode/utf8"
)

// ModHotkey is a binding declared by a mod's [Key...] sections.
// Read-only: no edits to .ini are made by the manager.
type ModHotkey struct {
    Section string   `json:"section"`
    Label   string   `json:"label"`
    Keys    []string `json:"keys"`
    Back    []string `json:"back"`
    Mode    string   `json:"mode"`
    File    string   `json:"file"`
    Condition string `json:"condition,omitempty"`
}

const maxHotkeyINIBytes = 2 << 20
const maxModHotkeys = 120
var hotkeyHeaderRE = regexp.MustCompile(`(?i)^\[\s*(key[^\]]+)\s*\]$`)
var camelWords = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func readableKeySection(section string) string {
    title := strings.TrimSpace(section)
    title = strings.TrimSpace(title[3:]) // section starts with Key
    title = strings.TrimLeft(title, " _-")
    title = camelWords.ReplaceAllString(title, "${1} ${2}")
    title = strings.ReplaceAll(title, "_", " ")
    title = strings.ReplaceAll(title, "-", " ")
    title = strings.Join(strings.Fields(title), " ")
    if title == "" { return "Atalho do mod" }
    if len([]rune(title)) > 80 { title = string([]rune(title)[:80]) }
    return title
}

func iniHotkeyText(data []byte) string {
    if len(data) >= 2 && ((data[0] == 0xff && data[1] == 0xfe) || (data[0] == 0xfe && data[1] == 0xff)) {
        little := data[0] == 0xff
        pairs := make([]uint16, 0, (len(data)-2)/2)
        for i := 2; i+1 < len(data); i += 2 {
            if little { pairs = append(pairs, uint16(data[i]) | uint16(data[i+1]) << 8) } else { pairs = append(pairs, uint16(data[i]) << 8 | uint16(data[i+1])) }
        }
        return string(utf16.Decode(pairs))
    }
    if !utf8.Valid(data) { return string(bytes.ToValidUTF8(data, []byte(" "))) }
    return strings.TrimPrefix(string(data), "\ufeff")
}

func stripINIComment(line string) string {
    // The delimiters have no escaping in regular ZZMI key declarations.
    // Ignore inline comments, preserving bare key names including spaces.
    if i := strings.IndexAny(line, ";#"); i >= 0 {
        return strings.TrimSpace(line[:i])
    }
    return strings.TrimSpace(line)
}

func parseKeybinds(data []byte, source string) []ModHotkey {
    text := iniHotkeyText(data)
    scanner := bufio.NewScanner(strings.NewReader(text))
    scanner.Buffer(make([]byte, 64*1024), maxHotkeyINIBytes+1)
    var all []ModHotkey
    var current *ModHotkey
    commit := func() {
        if current == nil || len(current.Keys) == 0 || len(all) >= maxModHotkeys { return }
        all = append(all, *current)
    }
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") { continue }
        if strings.HasPrefix(line, "[") {
            commit()
            current = nil
            if matches := hotkeyHeaderRE.FindStringSubmatch(line); len(matches) == 2 {
                section := strings.TrimSpace(matches[1])
                // The section header must have the Key prefix, not "Keyboard".
                if len(section) >= 3 && strings.EqualFold(section[:3], "Key") {
                    current = &ModHotkey{Section: section, Label: readableKeySection(section), Keys: []string{}, Back: []string{}, File: filepath.ToSlash(source)}
                }
            }
            continue
        }
        if current == nil { continue }
        pos := strings.IndexByte(line, '=')
        if pos < 1 { continue }
        field := strings.ToLower(strings.TrimSpace(line[:pos]))
        value := stripINIComment(line[pos+1:])
        if value == "" { continue }
        if len([]rune(value)) > 160 { value = string([]rune(value)[:160]) }
        switch field {
        case "key":
            if len(current.Keys) < 12 { current.Keys = append(current.Keys, value) }
        case "back":
            if len(current.Back) < 12 { current.Back = append(current.Back, value) }
        case "type":
            current.Mode = strings.ToLower(value)
        case "condition":
            current.Condition = value
        }
    }
    commit()
    return all
}

func readModHotkeys(iniPath, modRoot string, existing []ModHotkey) []ModHotkey {
    if len(existing) >= maxModHotkeys { return existing }
    st, err := os.Stat(iniPath)
    if err != nil || st.Size() > maxHotkeyINIBytes || st.Size() < 0 { return existing }
    data, err := os.ReadFile(iniPath)
    if err != nil { return existing }
    rel, err := filepath.Rel(modRoot, iniPath)
    if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) { return existing }
    for _, binding := range parseKeybinds(data, rel) {
        if len(existing) >= maxModHotkeys { break }
        // Keep independent declarations even if they share the same key.
        existing = append(existing, binding)
    }
    return existing
}

func isInterestingHotkey(section string) bool {
    return len(section) > 3 && strings.EqualFold(section[:3], "Key") && (unicode.IsUpper(rune(section[3])) || section[3] == '_' || section[3] == '-' || unicode.IsDigit(rune(section[3])))
}
