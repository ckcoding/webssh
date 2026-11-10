package core

import (
	"encoding/base64"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestGenerateClientTimesOutStalledHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		io.Copy(io.Discard, conn)
	}()
	client, err := BuildSSHClient("127.0.0.1", "test", "test", listener.Addr().(*net.TCPAddr).Port, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = client.generateClient(50 * time.Millisecond)
	var networkErr net.Error
	if !errors.As(err, &networkErr) || !networkErr.Timeout() {
		t.Fatalf("expected handshake timeout, got %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out handshake left its socket open")
	}
}

func TestBuildSSHClientValidatesPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{name: "default port", port: 0, wantErr: false},
		{name: "valid port", port: 2222, wantErr: false},
		{name: "negative port", port: -1, wantErr: true},
		{name: "too large port", port: 65536, wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client, err := BuildSSHClient("127.0.0.1", "root", "secret", tt.port, 0)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.port == 0 && client.Port != 22 {
				t.Fatalf("default port = %d, want 22", client.Port)
			}
		})
	}
}

func TestStrictHostKeyCheckingEnabled(t *testing.T) {
	t.Setenv("WEBSSH_STRICT_HOST_KEY", "")
	if strictHostKeyCheckingEnabled() {
		t.Fatalf("empty env should disable strict host key checking")
	}

	t.Setenv("WEBSSH_STRICT_HOST_KEY", "true")
	if !strictHostKeyCheckingEnabled() {
		t.Fatalf("true should enable strict host key checking")
	}
}

func TestDecodedMsgToSSHClientDefaultsZeroPort(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte(`{"username":"root","password":"secret","ipaddress":"127.0.0.1","port":0}`))

	client, err := DecodedMsgToSSHClient(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Port != 22 {
		t.Fatalf("port = %d, want 22", client.Port)
	}
}
