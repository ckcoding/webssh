package main

import "testing"

func TestLoadRuntimeConfigAllowsColonInPassword(t *testing.T) {
	oldAuthInfo := *authInfo
	oldUsername := username
	oldPassword := password
	defer func() {
		*authInfo = oldAuthInfo
		username = oldUsername
		password = oldPassword
	}()

	*authInfo = "admin:pass:with:colon"
	username = ""
	password = ""

	if !loadRuntimeConfig() {
		t.Fatalf("loadRuntimeConfig returned false")
	}
	if username != "admin" {
		t.Fatalf("username = %q, want admin", username)
	}
	if password != "pass:with:colon" {
		t.Fatalf("password = %q, want pass:with:colon", password)
	}
}

func TestTokenMatches(t *testing.T) {
	if !tokenMatches(sessionToken) {
		t.Fatalf("session token should match")
	}
	if tokenMatches(sessionToken + "x") {
		t.Fatalf("different token should not match")
	}
	if tokenMatches("") {
		t.Fatalf("empty token should not match")
	}
}

func TestConstantTimeStringEqual(t *testing.T) {
	if !constantTimeStringEqual("same", "same") {
		t.Fatalf("same strings should match")
	}
	if constantTimeStringEqual("same", "different") {
		t.Fatalf("different strings should not match")
	}
	if constantTimeStringEqual("", "different") {
		t.Fatalf("empty candidate should not match")
	}
}

func TestRedactRequestPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "redacts sensitive query keys",
			path: "/term?sshInfo=secret&sessionId=sid&token=abc&cols=80",
			want: "/term?cols=80&sessionId=REDACTED&sshInfo=REDACTED&token=REDACTED",
		},
		{
			name: "keeps ordinary query keys",
			path: "/file/list?path=/root",
			want: "/file/list?path=%2Froot",
		},
		{
			name: "redacts malformed query",
			path: "/term?sshInfo=%zz",
			want: "/term?REDACTED_QUERY",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := redactRequestPath(tt.path); got != tt.want {
				t.Fatalf("redactRequestPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
