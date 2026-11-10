package core

import (
	"errors"
	"testing"
)

func TestIsBrokenFileSessionError(t *testing.T) {
	if !isBrokenFileSessionError(errors.New("EOF")) {
		t.Fatalf("expected EOF to be treated as a broken file session")
	}
	if isBrokenFileSessionError(errors.New("permission denied")) {
		t.Fatalf("did not expect permission errors to invalidate the session")
	}
}
