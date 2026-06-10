package agent

import "testing"

func TestApplyEditUnique(t *testing.T) {
	got, n, err := applyEdit("alpha\nbeta\ngamma", "beta", "BETA", false)
	if err != nil || n != 1 {
		t.Fatalf("unexpected: %v %d", err, n)
	}
	if got != "alpha\nBETA\ngamma" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyEditNotFound(t *testing.T) {
	_, _, err := applyEdit("hello", "missing", "x", false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyEditNotUnique(t *testing.T) {
	_, _, err := applyEdit("foo bar foo", "foo", "x", false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyEditReplaceAll(t *testing.T) {
	got, n, err := applyEdit("foo bar foo", "foo", "x", true)
	if err != nil || n != 2 {
		t.Fatalf("unexpected: %v %d", err, n)
	}
	if got != "x bar x" {
		t.Fatalf("got %q", got)
	}
}
