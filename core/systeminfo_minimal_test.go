package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSystemInfoOnMinimalLinuxWithoutCPUUtilities(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell fixture")
	}
	dir := t.TempDir()
	for _, name := range []string{"nproc", "getconf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "uname"), []byte("#!/bin/sh\nprintf 'Linux\\n'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	cpuinfo := filepath.Join(dir, "cpuinfo")
	if err := os.WriteFile(cpuinfo, []byte("processor : 0\nprocessor : 1\nprocessor : 2\nprocessor : 3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
	command := strings.Replace(systemInfoCommand, "PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin", "PATH="+quote(dir+":"+os.Getenv("PATH")), 1)
	command = strings.ReplaceAll(command, "/proc/cpuinfo", quote(cpuinfo))
	output, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseSSHSystemInfoOutput(string(output))
	if err != nil {
		t.Fatal(err)
	}
	if parsed["cpu_cores"] != "4" {
		t.Fatalf("Missing CPU count without nproc/getconf: %s", parsed["cpu_cores"])
	}
}
