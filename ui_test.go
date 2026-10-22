package main

import (
	"strings"
	"testing"
)

func TestToolbarActionButtonsKeepWhiteIcons(t *testing.T) {
	css := chromeInjectionJS

	checks := []string{
		"#__sparrow_chrome #btn-app-lock { width: 26px; height: 26px; flex: 0 0 26px; display: flex; align-items: center; justify-content: center; border: 0; border-radius: 6px; padding: 4px; background: transparent; color: #fff !important; cursor: pointer; }",
		"#__sparrow_chrome #btn-app-lock svg *, #__sparrow_chrome #qr-share-btn svg *, #__sparrow_chrome #btn-app-lock svg, #__sparrow_chrome #qr-share-btn svg { fill: currentColor !important; stroke: currentColor !important; }",
		"#__sparrow_chrome #qr-share-btn { width: 26px; height: 26px; padding: 4px; border: 0; border-radius: 6px; display: flex; align-items: center; justify-content: center; cursor: pointer; color: #fff !important; background: transparent; flex-shrink: 0; transition: background 0.16s ease, color 0.16s ease, box-shadow 0.16s ease; }",
	}

	for _, check := range checks {
		if !strings.Contains(css, check) {
			t.Fatalf("toolbar button CSS missing white icon rule: %q", check)
		}
	}
}
