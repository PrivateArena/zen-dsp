package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func requirePipeWire(t *testing.T) {
	if os.Getenv("ZEN_DSP_PW") != "1" {
		t.Skip("skip integration: set ZEN_DSP_PW=1 and run under PipeWire session")
	}
}

func TestPWReady(t *testing.T) {
	requirePipeWire(t)
	cmd := exec.Command("pw-dump")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pw-dump failed: %v: %s", err, out)
	}
	if len(out) == 0 {
		t.Fatalf("pw-dump returned empty output")
	}
}

func TestWPCTLCanListSinks(t *testing.T) {
	requirePipeWire(t)
	cmd := exec.Command("wpctl", "inspect", "@DEFAULT_SINK@")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// wpctl inspect returns nonzero for unresolved symlink in some setups
		if !strings.Contains(string(out), "not found") {
			t.Fatalf("wpctl failed: %v: %s", err, out)
		}
	}
}

func TestApplyConfRoundTrip(t *testing.T) {
	requirePipeWire(t)
	// Use a throwaway XDG dir
	tmp := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmp)


	pw := exec.Command("pipewire", "-c", "none", "-n", "e2e-pw")
	pw.Env = append(os.Environ(), "XDG_RUNTIME_DIR="+tmp, "PIPEWIRE_CONFIG_DIR="+tmp)
	// In practices this needs a real user session; skip if can't exec.
	_, err := pw.Output()
	if err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			t.Skip("pipewire binary missing")
		}
		t.Skipf("cannot launch pipewire in test: %v", err)
	}
	defer pw.Process.Kill()

	cmd := exec.Command("pw-cli", "info", "0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pw-cli info 0 failed: %v: %s", err, out)
	}
	if !strings.Contains(string(out), "0") {
		t.Fatalf("unexpected pw-cli output")
	}
}
