package core

import "testing"

func TestParseSSHSystemInfoOutput(t *testing.T) {
	output := `hostname=web-01
os=Ubuntu 24.04.2 LTS
kernel=Linux 6.8.0-57-generic x86_64 GNU/Linux
load_average=0.10 0.20 0.30
cpu_cores=8
cpu_usage=12.50
mem_total_kb=8388608
mem_available_kb=4194304
disk_mount=/
disk_total_kb=52428800
disk_available_kb=26214400
disk_used_kb=26214400
uptime_seconds=93784
`

	parsed, err := parseSSHSystemInfoOutput(output)
	if err != nil {
		t.Fatalf("parseSSHSystemInfoOutput() error = %v", err)
	}
	if parsed["hostname"] != "web-01" {
		t.Fatalf("expected hostname web-01, got %q", parsed["hostname"])
	}
	if parseInt(parsed["cpu_cores"]) != 8 {
		t.Fatalf("expected cpu_cores = 8, got %q", parsed["cpu_cores"])
	}
	if parseFloat(parsed["cpu_usage"]) != 12.50 {
		t.Fatalf("expected cpu_usage = 12.50, got %q", parsed["cpu_usage"])
	}
	if parseInt64(parsed["mem_total_kb"]) != 8388608 {
		t.Fatalf("expected mem_total_kb = 8388608, got %q", parsed["mem_total_kb"])
	}
	if parseInt64(parsed["disk_total_kb"]) != 52428800 {
		t.Fatalf("expected disk_total_kb = 52428800, got %q", parsed["disk_total_kb"])
	}
}

func TestHumanizeUptime(t *testing.T) {
	if got := humanizeUptime(93784); got != "1d 2h 3m" {
		t.Fatalf("expected uptime 1d 2h 3m, got %q", got)
	}
	if got := humanizeUptime(0); got != "" {
		t.Fatalf("expected empty uptime for 0, got %q", got)
	}
}

func TestFormatIPLocation(t *testing.T) {
	got := formatIPLocation(ipWhoisResponse{
		City:        "Los Angeles",
		Region:      "California",
		Country:     "United States",
		CountryCode: "us",
	})
	want := "Los Angeles, California, United States"
	if got != want {
		t.Fatalf("expected location %q, got %q", want, got)
	}
}

func TestCountryFlagEmoji(t *testing.T) {
	if got := countryFlagEmoji("JP"); got != "🇯🇵" {
		t.Fatalf("expected JP flag, got %q", got)
	}
	if got := countryFlagEmoji("unknown"); got != "" {
		t.Fatalf("expected empty flag for invalid code, got %q", got)
	}
}
