package main

import (
    "io/fs"
    "os"
    "path/filepath"
    "strings"
)

// componentFolderName recognizes common body-part subfolders shipped as
// independent .ini files within ONE outfit, including numbered alternatives.
// Avoid general numeric-stripping on outfit names: only known component bases
// qualify, so 'School Outfit 2' remains a separate mod.
func componentFolderName(name string) bool {
    name = strings.ToLower(strings.TrimSpace(prettyName(name)))
    base := strings.TrimRight(name, "0123456789")
    base = strings.TrimRight(base, " _-")
    switch base {
    case "body", "face", "hair", "head", "weapon", "wuqi", "tex",
        "eyes", "arms", "legs", "shoes", "tail", "clothes",
        "cloth", "accessory", "accessories":
        return true
    }
    return false
}

func packageSupportFolder(name string) bool {
    base := strings.ToLower(strings.TrimSpace(prettyName(name)))
    switch base {
    case "toggles", "resources", "textures", "meshes":
        return true
    }
    return false
}

// looksLikeSingleModPackage identifies multipart outfits whose .ini files
// live in component subfolders, e.g.
// Mods/Ye Shunguang/YeShunguang-Purple Red Demon Spirit v2.0/
//     body/, face/, hair/, hair2/, tex/, wuqi/, wuqi2/.
// Do not consolidate an agent's collection of unrelated outfits.
func looksLikeSingleModPackage(parent string, entries []os.DirEntry) bool {
    name := strings.ToLower(prettyName(filepath.Base(parent)))
    if knownWardrobeAgent(prettyName(filepath.Base(parent))) {
        return false
    }
    componentParts, supportDirs, otherDirs := 0, 0, 0
    hasPackagePreview := false
    for _, item := range entries {
        if !item.IsDir() {
            if item.Type()&os.ModeSymlink != 0 { continue }
            filename := strings.ToLower(item.Name())
            ext := filepath.Ext(filename)
            base := strings.TrimSuffix(filename, ext)
            if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp" {
                switch base {
                case "preview", "preview_image", "previewimage", "thumbnail", "thumb", "cover":
                    hasPackagePreview = true
                }
            }
            continue
        }
        child := strings.ToLower(prettyName(item.Name()))
        if child == name {
            return true // duplicated archive wrapper, e.g. RoxyOveralls/RoxyOveralls
        }
        if componentFolderName(child) {
            componentParts++
        } else if packageSupportFolder(child) {
            supportDirs++
        } else {
            otherDirs++
        }
    }
    // Pure collections of conventional body/face/weapon/etc. subfolders are
    // one mod even without a preview at the package root.
    if otherDirs == 0 && (componentParts >= 2 || supportDirs >= 1) {
        return true
    }
    // Named skin directories + ancillary hair/body parts need additional
    // evidence so we don't accidentally merge several unrelated outfits.
    return hasPackagePreview && (componentParts+supportDirs) >= 1 && otherDirs >= 1
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
