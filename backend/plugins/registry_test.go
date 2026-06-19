package plugins

import (
	"testing"

	"github.com/enough/enough/backend/config"
)

func TestIsValidNamespace(t *testing.T) {
	cases := []struct {
		ns   string
		want bool
	}{
		{"my-plugin", true},
		{"plugin123", true},
		{"Plugin_Name", true},
		{"my:plugin", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsValidNamespace(tc.ns); got != tc.want {
			t.Errorf("IsValidNamespace(%q) = %v, want %v", tc.ns, got, tc.want)
		}
	}
}

func TestParseQualifiedName(t *testing.T) {
	cases := []struct {
		input string
		wantN string
		wantB string
	}{
		{"ns:skill", "ns", "skill"},
		{"bare-skill", "", "bare-skill"},
		{"ns:sub:skill", "ns", "sub:skill"},
	}
	for _, tc := range cases {
		gotN, gotB := ParseQualifiedName(tc.input)
		if gotN != tc.wantN || gotB != tc.wantB {
			t.Errorf("ParseQualifiedName(%q) = (%q, %q), want (%q, %q)", tc.input, gotN, gotB, tc.wantN, tc.wantB)
		}
	}
}

func TestIsPluginDisabled(t *testing.T) {
	cfg := config.Runtime{
		Plugins: config.PluginsSettings{
			Disabled: []string{"disabled-plugin"},
		},
	}
	if !IsPluginDisabled("disabled-plugin", cfg) {
		t.Error("expected disabled-plugin to be disabled")
	}
	if IsPluginDisabled("enabled-plugin", cfg) {
		t.Error("expected enabled-plugin to be enabled")
	}
}
