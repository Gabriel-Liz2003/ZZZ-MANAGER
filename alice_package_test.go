package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The real-world package layout is:
// Mods/Alice/<Chinese outfit title>/preview.png,
// Mods/Alice/<Chinese outfit title>/aliceskin/*.ini,
// Mods/Alice/<Chinese outfit title>/hair/*.ini.
func TestAliceSuitNamedSkinAndHairAreOneMod(t *testing.T) {
	root := makeMods(t)
	bundle := "Alice/爱丽丝兔女郎NSFW Suit动画版"
	writeModFile(t, root, bundle+"/preview.png", "image bytes")
	writeModFile(t, root, bundle+"/作者的话(read me).txt", "installation notes")
	writeModFile(t, root, bundle+"/aliceskin/alice1.ini", "[KeyMenu]\nkey = l\ntype = cycle\n[TextureOverrideA]\nhash = deadbeef\n")
	writeModFile(t, root, bundle+"/aliceskin/alice2.ini", "[TextureOverrideB]\nhash = 11111111\n")
	writeModFile(t, root, bundle+"/hair/hair.ini", "[TextureOverrideHair]\nhash = deadbeef\n[TextureOverrideHair2]\nhash = 22222222\n")

	mods, conflicts, err := scanMods(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatalf("the skin and hair must be ONE mod, got %d: %#v", len(mods), mods)
	}
	m := mods[0]
	if m.ID != bundle || m.RelPath != bundle || m.path != filepath.Join(root, filepath.FromSlash(bundle)) {
		t.Fatalf("the parent outfit must be the package root: %+v", m)
	}
	if m.Character != "Alice" || !m.Enabled || m.IniCount != 3 || !m.Preview {
		t.Fatalf("wrong agent/state/INI/preview: %+v", m)
	}
	if len(m.hashes) != 3 || !m.hashes["deadbeef"] || !m.hashes["11111111"] || !m.hashes["22222222"] {
		t.Fatalf("missing or duplicated component hashes: %#v", m.hashes)
	}
	if len(conflicts) != 0 || m.Conflicts != 0 {
		t.Fatalf("hashes shared by parts of ONE mod are not conflicts: %#v", conflicts)
	}
	if len(m.Hotkeys) != 1 || m.Hotkeys[0].Keys[0] != "l" || m.Hotkeys[0].File != "aliceskin/alice1.ini" {
		t.Fatalf("nested hotkey not indexed: %#v", m.Hotkeys)
	}
	for _, oldPart := range []string{bundle + "/hair", bundle + "/aliceskin"} {
		if findModForSavedProfile(mods, oldPart) != m {
			t.Fatalf("the previous component ID %q must resolve to the merged package", oldPart)
		}
	}
	if err := toggleMod(m, false); err != nil {
		t.Fatal(err)
	}
	disabled := filepath.Join(root, "Alice", "DISABLED_爱丽丝兔女郎NSFW Suit动画版")
	for _, relative := range []string{"aliceskin/alice1.ini", "hair/hair.ini", "preview.png"} {
		if _, err := os.Stat(filepath.Join(disabled, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("disabling must preserve %s with the package: %v", relative, err)
		}
	}
	mods, _, err = scanMods(root, nil)
	if err != nil || len(mods) != 1 || mods[0].Enabled {
		t.Fatalf("disabled outfit must still count as one mod: %v, %+v", err, mods)
	}
	if err := toggleMod(mods[0], true); err != nil {
		t.Fatal(err)
	}
	mods, _, err = scanMods(root, nil)
	if err != nil || len(mods) != 1 || !mods[0].Enabled {
		t.Fatalf("reenabled outfit must still count as one mod: %v, %+v", err, mods)
	}
}

func TestAliceOutfitBundleDoesNotCombineOtherOutfits(t *testing.T) {
	root := makeMods(t)
	bundle := "Alice/爱丽丝兔女郎NSFW Suit动画版"
	writeModFile(t, root, bundle+"/preview.png", "image bytes")
	writeModFile(t, root, bundle+"/aliceskin/suit.ini", "hash = deadbeef\n")
	writeModFile(t, root, bundle+"/hair/hair.ini", "hash = 22222222\n")
	writeModFile(t, root, "Alice/Other Outfit/mod.ini", "hash = 33333333\n")
	writeModFile(t, root, "Alice/Second Outfit/mod.ini", "hash = 44444444\n")
	// Even if the agent category has an image, it must remain a collection.
	writeModFile(t, root, "Alice/preview.png", "generic agent image")
	mods, conflicts, err := scanMods(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 3 || len(conflicts) != 0 {
		t.Fatalf("expected suit (with hair) + TWO other independent outfits: %d, %#v", len(mods), conflicts)
	}
	seen := map[string]bool{}
	for _, m := range mods {
		seen[m.ID] = true
		if m.Character != "Alice" {
			t.Fatalf("different agent inferred: %#v", m)
		}
	}
	for _, id := range []string{bundle, "Alice/Other Outfit", "Alice/Second Outfit"} {
		if !seen[id] {
			t.Fatalf("outfit %s should remain separate: %#v", id, mods)
		}
	}
	if seen["Alice"] || seen[bundle+"/hair"] || seen[bundle+"/aliceskin"] {
		t.Fatalf("agent directory or individual components were misdetected: %#v", seen)
	}
}

func TestUnmarkedAmbiguousAliceFoldersRemainSeparate(t *testing.T) {
	root := makeMods(t)
	// Do not guess: without a package preview, generic child directories
	// might be completely separate user-installed mods.
	writeModFile(t, root, "Alice/Ambiguous collection/customskin/skin.ini", "hash = 11111111\n")
	writeModFile(t, root, "Alice/Ambiguous collection/hair/hair.ini", "hash = 22222222\n")
	mods, _, err := scanMods(root, nil)
	if err != nil || len(mods) != 2 {
		t.Fatalf("ambiguous folders without package evidence must stay separate: %v, %+v", err, mods)
	}
}
