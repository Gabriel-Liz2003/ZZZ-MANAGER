package main

import (
    "os"
    "path/filepath"
    "testing"
)

func TestYeShunguangRedDemonSpiritComponentsAreOneMod(t *testing.T) {
    root := makeMods(t)
    bundle := "Ye Shunguang/YeShunguang-Purple Red Demon Spirit v2.0"
    parts := []string{"body", "face", "hair", "hair2", "tex", "wuqi", "wuqi2"}
    for _, part := range parts {
        hash := "11111111"
        if part == "face" { hash = "22222222" }
        if part == "hair" { hash = "33333333" }
        data := "[TextureOverridePart]\nhash = " + hash + "\n"
        if part == "body" {
            data = "[Constants]\nglobal persist $dress = 1\n[KeySwapDress]\nkey = y\n" + data
        }
        if part == "hair2" {
            data = "[KeySwapHair]\nkey = VK_NUMPAD6\n" + data
        }
        writeModFile(t, root, bundle+"/"+part+"/"+part+".ini", data)
    }
    writeModFile(t, root, bundle+"/body/extra.ini", "[TextureOverrideAnother]\nhash = 44444444\n")
    mods, conflicts, err := scanMods(root, nil)
    if err != nil { t.Fatal(err) }
    if len(mods) != 1 {
        t.Fatalf("Expected a single Ye Shunguang mod (not seven component cards), got %d: %+v", len(mods), mods)
    }
    m := mods[0]
    if m.ID != bundle || m.RelPath != bundle || m.Name != "YeShunguang-Purple Red Demon Spirit v2.0" {
        t.Fatalf("Wrong package identity: %+v", m)
    }
    if m.Character != "Ye Shunguang" || m.IniCount != 8 || !m.Enabled {
        t.Fatalf("Wrong character/count/state: %+v", m)
    }
    if m.PersistentVariables != 1 || len(m.Hotkeys) != 2 {
        t.Fatalf("Nested commands and presets must be indexed once: %+v", m)
    }
    if len(m.hashes) != 4 || len(conflicts) != 0 || m.Conflicts != 0 {
        t.Fatalf("Parts of one package must not conflict with each other: %v %v", len(m.hashes), conflicts)
    }
    for _, part := range parts {
        if old := findModForSavedProfile(mods, bundle+"/"+part); old != m {
            t.Fatalf("Legacy profile for %s did not resolve to the parent package", part)
        }
    }
    if err := toggleMod(m, false); err != nil { t.Fatal(err) }
    disabled := filepath.Join(root, "Ye Shunguang", "DISABLED_YeShunguang-Purple Red Demon Spirit v2.0")
    for _, part := range parts {
        if _, err := os.Stat(filepath.Join(disabled, part, part+".ini")); err != nil {
            t.Fatalf("Disabling package lost %s: %v", part, err)
        }
    }
    mods, _, err = scanMods(root, nil)
    if err != nil || len(mods) != 1 || mods[0].Enabled {
        t.Fatalf("Disabled package must still be one mod: %v %+v", err, mods)
    }
    if err := toggleMod(mods[0], true); err != nil { t.Fatal(err) }
    mods, _, err = scanMods(root, nil)
    if err != nil || len(mods) != 1 || !mods[0].Enabled {
        t.Fatalf("Reenabled package must stay one mod: %v %+v", err, mods)
    }
}

func TestYeShunguangPackageConflictsOnlyWithSeparateOutfit(t *testing.T) {
    root := makeMods(t)
    bundle := "Ye Shunguang/YeShunguang-Purple Red Demon Spirit v2.0"
    writeModFile(t,root,bundle+"/body/a.ini","[TextureOverrideBody]\nhash=deadbeef\n")
    writeModFile(t,root,bundle+"/hair2/hair.ini","[TextureOverrideHair]\nhash=deadbeef\n")
    writeModFile(t,root,bundle+"/wuqi2/weapon.ini","[TextureOverrideWeapon]\nhash=12345678\n")
    writeModFile(t,root,"Ye Shunguang/Another Outfit/other.ini","[TextureOverride]\nhash=deadbeef\n")
    mods, conflicts, err := scanMods(root,nil)
    if err != nil || len(mods)!=2 {
        t.Fatalf("Must not merge other outfits in the agent directory: %v %+v",err,mods)
    }
    if len(conflicts)!=1 || len(conflicts[0].ModIDs)!=2 {
        t.Fatalf("There should be one real cross-mod conflict, not duplicates: %+v",conflicts)
    }
}

func TestAgentRootNeverGetsCombinedWithMultipartComponentFolders(t *testing.T) {
    root := makeMods(t)
    // Agent directories are collections even if some directories have
    // component-like names (a mod named "hair" or a "resources" folder).
    writeModFile(t,root,"Ye Shunguang/body/body.ini","hash = aaaabbbb")
    writeModFile(t,root,"Ye Shunguang/face/face.ini","hash = ccccdddd")
    writeModFile(t,root,"Ye Shunguang/Other Outfit/skin.ini","hash = ddddcccc")
    mods,_,err:=scanMods(root,nil)
    if err!=nil || len(mods)!=3 {
        t.Fatalf("Never consolidate the agent's collection: %v %+v",err,mods)
    }
    for _,m:=range mods {
        if m.ID=="Ye Shunguang" {t.Fatal("Agent category itself must not be a mod")}
    }
}

func TestNumberedComponentsAreRecognizedOnlyWhenKnown(t *testing.T) {
    for _,name:=range []string{"hair", "hair2","hair_3","wuqi2","tex","body","face","weapon","DISABLED_hair2"} {
        if !componentFolderName(name) {t.Errorf("Unrecognized component %q",name)}
    }
    for _,name:=range []string{"Outfit 2","School Outfit 3","customskin","FancyHair","hairpiece","Facepaint"} {
        if componentFolderName(name) {t.Errorf("Unrelated folder recognized as component: %q",name)}
    }
}
