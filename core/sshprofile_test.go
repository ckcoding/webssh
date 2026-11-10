package core

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSSHProfilesEncryptionRoundTrip(t *testing.T) {
	t.Setenv("WEBSSH_SECRET_KEY", "test-secret")

	profiles := []SavedSSHProfile{
		{
			Host:      "127.0.0.1",
			Username:  "root",
			Port:      22,
			Password:  "secret",
			LoginType: 0,
		},
	}

	content, err := encodeSSHProfiles(profiles)
	if err != nil {
		t.Fatalf("encodeSSHProfiles failed: %v", err)
	}
	if bytes.Contains(content, []byte("secret")) {
		t.Fatalf("encrypted profile content should not contain plaintext password")
	}

	var encrypted encryptedSSHProfilesFile
	if err := json.Unmarshal(content, &encrypted); err != nil {
		t.Fatalf("encrypted profile is not json object: %v", err)
	}
	if encrypted.Algorithm != "AES-256-GCM" {
		t.Fatalf("algorithm = %q, want AES-256-GCM", encrypted.Algorithm)
	}

	decoded, err := decodeSSHProfiles(content)
	if err != nil {
		t.Fatalf("decodeSSHProfiles failed: %v", err)
	}
	if len(decoded) != 1 || decoded[0].Password != "secret" {
		t.Fatalf("decoded profiles = %#v", decoded)
	}
}

func TestSSHProfilesPlaintextCompatibility(t *testing.T) {
	t.Setenv("WEBSSH_SECRET_KEY", "")

	content := []byte(`[{"host":"127.0.0.1","username":"root","port":22,"password":"secret"}]`)
	decoded, err := decodeSSHProfiles(content)
	if err != nil {
		t.Fatalf("decode plaintext failed: %v", err)
	}
	if len(decoded) != 1 || decoded[0].Password != "secret" {
		t.Fatalf("decoded profiles = %#v", decoded)
	}

	encoded, err := encodeSSHProfiles(decoded)
	if err != nil {
		t.Fatalf("encode plaintext failed: %v", err)
	}
	if !bytes.Contains(encoded, []byte("secret")) {
		t.Fatalf("plaintext mode should keep compatible json format")
	}
}
