package main

import (
    "io/fs"
    "os"
    "path/filepath"
    "strings"
)

// looksLikeSingleModPackage recognizes imported mod archives that have one or
// more internal folders (e.g. RoxyOveralls/RoxyOveralls/toggles and resources).
// Only a directory with package-layout evidence is grouped. Generic character
// collection directories remain traversable and retain independent mods.
func looksLikeSingleModPackage(parent string, entries []os.DirEntry) bool {
    name := strings.ToLower(prettyName(filepath.Base(parent)))
    // Some modders split a single character replacement into component
    // directories containing independent INIs. These must be toggled together.
    // Do not mistake a character's collection of several outfits for one mod.
    componentParts := 0
    otherDirs := 0
    for _, item := range entries {
        if !item.IsDir() { continue }
        child := strings.ToLower(prettyName(item.Name()))
        if child == name {
            return true // common duplicate wrapper from ZIP/RAR
        }
        switch child {
        case "toggles", "resources", "textures", "meshes":
            // An explicit support directory indicates a packaged mod. A
            // package with only "resources" still follows the old behavior.
            return true
        case "body", "face", "weapon", "hair", "head", "accessories":
            componentParts++
        default:
            otherDirs++
        }
    }
    // Example: Mods/Soldier 0-Anby-yyzj-/{body,face,weapon}/*.ini.
    // A character collection containing unrelated outfit subdirectories
    // remains traversable, preserving independent mods and their switches.
    return componentParts >= 2 && otherDirs == 0
}

// anyModINI is used only for directories with evidence of a package layout;
// avoid grouping a general directory of unrelated skins.
func anyModINI(dir string) bool {
    found := false
    _ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
        if err != nil { return nil }
        if d.IsDir() {
            if p != dir && (strings.HasPrefix(d.Name(), ".") ||
                strings.EqualFold(d.Name(), "ShaderFixes") ||
                strings.EqualFold(d.Name(), "ShaderCache") ||
                strings.EqualFold(d.Name(), "BufferValues")) {
                return filepath.SkipDir
            }
            return nil
        }
        if strings.EqualFold(filepath.Ext(d.Name()), ".ini") {
            found = true
            return fs.SkipAll
        }
        return nil
    })
    return found
}

// nestedModPartDisabled means an ancillary directory has its own DISABLED_
// prefix. Never treat its hashes or key assignments as active in the parent.
func nestedModPartDisabled(modRoot, filePath string) bool {
    rel, err := filepath.Rel(modRoot, filepath.Dir(filePath))
    if err != nil { return true }
    if rel == "." { return false }
    for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
        if isDisabled(part) { return true }
    }
    return false
}

// inheritedModSetting reads old per-component metadata after the scanner
// consolidates body/face/weapon into one package. Explicit package values
// always win. Conflicting child values are deliberately not guessed.
func inheritedModSetting(values map[string]string, rootID string) string {
    if value := values[rootID]; value != "" {
        return value
    }
    prefix := rootID + "/"
    found := ""
    for id, value := range values {
        if !strings.HasPrefix(id, prefix) || value == "" {
            continue
        }
        if found != "" && found != value {
            return "" // ambiguous old settings; let the user reclassify
        }
        found = value
    }
    return found
}

// findModForSavedProfile keeps presets made with older versions usable after
// internal "resources"/"toggles" entries are consolidated into their package.
// A saved child path maps to the parent only when that parent is now a real mod.
func findModForSavedProfile(mods []*Mod, oldID string) *Mod {
    if match := findMod(mods, oldID); match != nil { return match }
    for _, mod := range mods {
        if strings.HasPrefix(oldID, mod.ID+"/") { return mod }
    }
    return nil
}
