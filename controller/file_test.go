package controller

import "testing"

func TestBuildCreateTargetPath(t *testing.T) {
	got, err := buildCreateTargetPath("/var/www///", "notes.txt")
	if err != nil {
		t.Fatalf("buildCreateTargetPath() error = %v", err)
	}
	if got != "/var/www/notes.txt" {
		t.Fatalf("expected /var/www/notes.txt, got %q", got)
	}
}

func TestBuildCreateTargetPathRejectsInvalidName(t *testing.T) {
	cases := []string{"", " ", ".", "..", "a/b", "a\\b"}
	for _, input := range cases {
		if _, err := buildCreateTargetPath("/tmp", input); err == nil {
			t.Fatalf("expected error for %q, got nil", input)
		}
	}
}
