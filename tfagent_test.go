//go:build windows

package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// This helper test is invoked in a subprocess to run main() directly.
func TestInvokeMain(t *testing.T) {
	if os.Getenv("RUN_MAIN") != "1" {
		t.Skip("helper subprocess")
	}
	// Reset the default FlagSet so main's config.ParseFlags() can define/parse cleanly.
	flag.CommandLine = flag.NewFlagSet("tfagent", flag.ExitOnError)

	// If PRTCONF=1, simulate passing -prtconf to main.
	if os.Getenv("PRTCONF") == "1" {
		os.Args = []string{"tfagent", "-prtconf"}
	} else {
		os.Args = []string{"tfagent"}
	}

	main() // will os.Exit() on most paths; in subprocess, that's fine.
}

// --- Parent tests that spawn the subprocess ---

func TestMain_PrintConfigAndExitZero_WithPrtConf(t *testing.T) {
	// Create a ProgramData-like folder with a valid config.yaml
	tmp := t.TempDir()

	// main.GetConfigFile("TFAgent") will look in:
	//   %ProgramData%\TFAgent\config.yaml
	appDir := filepath.Join(tmp, "TFAgent")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create a valid config.yaml
	// Keep it minimal but must pass ValidateConfig:
	// - at least one transfer with a real source_directory
	srcDir := t.TempDir()
	yaml := `
logfile: ""
loglevel: "info"
logtoconsole: true
service_heartbeat: false
delay: 1
transfers:
  - name: "t1"
    source_directory: "` + filepath.ToSlash(srcDir) + `"
    remotepath: "/tmp"
    streaming: false
    transfertype: "sftp"
    username: "user"
    password: "pass"
    server: "127.0.0.1"
    port: "22"
    filter: ""
    archive_dest: ""
    action_on_success: "none"
    action_on_fail: "none"
    fail_dest: ""
`
	cfgPath := filepath.Join(appDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Run the test binary as a subprocess, instructing it to call main()
	cmd := exec.Command(os.Args[0], "-test.run", "TestInvokeMain", "-test.v")
	cmd.Env = append(os.Environ(),
		"RUN_MAIN=1",       // tells helper to call main()
		"PRTCONF=1",        // makes helper pass -prtconf to main
		"ProgramData="+tmp, // where main will look for config.yaml
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		t.Fatalf("expected exit code 0, got error: %v; output:\n%s", err, out.String())
	}

	// Should have printed YAML (best-effort check)
	if !strings.Contains(out.String(), "transfers:") {
		t.Fatalf("expected printed YAML config, got:\n%s", out.String())
	}
}

func TestMain_InvalidConfig_ExitCodeOne(t *testing.T) {
	tmp := t.TempDir()
	appDir := filepath.Join(tmp, "TFAgent")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Invalid config: no transfers → ValidateConfig should fail
	yaml := `
logfile: ""
loglevel: "info"
logtoconsole: true
service_heartbeat: false
delay: 1
transfers: []
`
	cfgPath := filepath.Join(appDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run", "TestInvokeMain", "-test.v")
	cmd.Env = append(os.Environ(),
		"RUN_MAIN=1",
		// no PRTCONF -> main should go through normal validation path
		"ProgramData="+tmp,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit due to invalid config; output:\n%s", out.String())
	}

	// On Windows, *exec.ExitError is returned for non-zero exit codes.
	// We don't assert exact code, just that it's non-zero and error mentions invalid configuration.
	if !strings.Contains(out.String(), "Invalid configuration") &&
		!strings.Contains(out.String(), "no transfers defined") {
		t.Logf("subprocess output:\n%s", out.String())
		t.Fatalf("expected validation failure message in output")
	}
}
