package main

import (
	"strings"
	"testing"
)

func TestCustomBrowserColorUpdatesWindowControlBackgrounds(t *testing.T) {
	checks := []string{
		"background: var(--shell-bg) !important;",
		"background: var(--window-bar) !important; color: var(--chrome-subtle)",
		"border: 0 !important; background: var(--window-bar) !important; color: var(--chrome-muted)",
		".window-control:hover { background: color-mix(in srgb, var(--chrome-accent) 12%, var(--window-bar)) !important;",
		".window-control.close:hover { background: #c42b1c !important;",
	}

	for _, check := range checks {
		if !strings.Contains(chromeInjectionJS, check) {
			t.Fatalf("custom browser color CSS missing window control background rule: %q", check)
		}
	}
}

func TestTabCloseButtonIsLargerAndTurnsRedOnHover(t *testing.T) {
	checks := []string{
		".tab-chip .tab-close { width: 24px; height: 24px;",
		"font-size: 18px;",
		".tab-chip .tab-close:hover { background: #c42b1c !important; color: #fff !important; }",
	}

	for _, check := range checks {
		if !strings.Contains(chromeInjectionJS, check) {
			t.Fatalf("tab close button CSS missing size or hover rule: %q", check)
		}
	}
}
