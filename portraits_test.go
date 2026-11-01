package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestBundledAgentPortraits(t *testing.T) {
	for _, name := range []string{"Belle", "Wise", "Remielle", "Sigrid", "Claret", "Roxy", "Phoenix", "Severian"} {
		t.Run(name, func(t *testing.T) {
			path := fmt.Sprintf("assets/portraits/%s.png", name)
			art, err := embedded.ReadFile(path)
			if err != nil {
				t.Fatalf("bundled art missing: %v", err)
			}
			if len(art) < 512 || !bytes.HasPrefix(art, []byte{137, 'P', 'N', 'G', 13, 10, 26, 10}) {
				t.Fatalf("invalid embedded PNG art: %s", path)
			}
		})
	}
}
