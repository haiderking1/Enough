package skills

import "testing"

func TestResolveSkillLookupName(t *testing.T) {
	cases := map[string]string{
		"enough-agent": "enough-agent",
		"enough":       "enough-agent",
		"hermes-agent": "enough-agent",
		"  Enough  ":  "enough-agent",
		"other-skill":  "other-skill",
	}
	for in, want := range cases {
		if got := ResolveSkillLookupName(in); got != want {
			t.Fatalf("ResolveSkillLookupName(%q) = %q, want %q", in, got, want)
		}
	}
}
