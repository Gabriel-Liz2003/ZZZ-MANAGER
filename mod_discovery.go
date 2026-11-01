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
    for _, item := range entries {
        if !item.IsDir() { continue }
        child := strings.ToLower(prettyName(item.Name()))
        if child == name {
            return true // common duplicate wrapper from ZIP/RAR
        }
        // "toggles" and "resources" are supporting folders, not mods.
        if child == "toggles" || child == "resources" || child == "textures" || child == "meshes" {
            return true
        }
    }
    return false
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
