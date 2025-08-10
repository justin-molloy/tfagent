package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ValidateConfig checks the loaded config and returns a single error describing all issues.
func ValidateConfig(cfg *ConfigData) error {
	var errs multiErr

	// ---- top-level checks ----
	if cfg == nil {
		return errors.New("config is nil")
	}
	if !isValidLogLevel(cfg.LogLevel) && cfg.LogLevel != "" {
		errs.addf("invalid loglevel %q (allowed: debug, info, warn, error)", cfg.LogLevel)
	}
	if len(cfg.Transfers) == 0 {
		errs.addf("no transfers defined")
	}

	// ---- per-transfer checks ----
	seenNames := map[string]struct{}{}
	for i := range cfg.Transfers {
		t := &cfg.Transfers[i]
		prefix := fmt.Sprintf("transfer[%d]", i)
		if strings.TrimSpace(t.Name) == "" {
			errs.addf("%s: name is required", prefix)
		} else {
			if _, dup := seenNames[t.Name]; dup {
				errs.addf("%s: duplicate name %q", prefix, t.Name)
			}
			seenNames[t.Name] = struct{}{}
		}

		// SourceDirectory
		if strings.TrimSpace(t.SourceDirectory) == "" {
			errs.addf("%s: source_directory is required", prefix)
		} else {
			if !isDir(t.SourceDirectory) {
				errs.addf("%s: source_directory %q does not exist or is not a directory", prefix, t.SourceDirectory)
			}
			// Optional: normalize to absolute path to avoid surprises later.
			if abs, err := filepath.Abs(t.SourceDirectory); err == nil {
				t.SourceDirectory = abs
			}
		}

		// TransferType
		switch strings.ToLower(strings.TrimSpace(t.TransferType)) {
		case "sftp":
			// Required fields for SFTP
			if strings.TrimSpace(t.Username) == "" {
				errs.addf("%s: username is required for SFTP", prefix)
			}
			if strings.TrimSpace(t.Server) == "" {
				errs.addf("%s: server is required for SFTP", prefix)
			}
			if !isValidPort(t.Port) {
				errs.addf("%s: port %q must be an integer 1-65535 for SFTP", prefix, t.Port)
			}
			if strings.TrimSpace(t.RemotePath) == "" {
				errs.addf("%s: remotepath is required for SFTP", prefix)
			}
			// Auth: require at least one of PrivateKey or Password
			if strings.TrimSpace(t.PrivateKey) == "" && strings.TrimSpace(t.Password) == "" {
				errs.addf("%s: either privatekey or password must be provided for SFTP", prefix)
			}
			if strings.TrimSpace(t.PrivateKey) != "" && !isFile(t.PrivateKey) {
				errs.addf("%s: privatekey file %q not found or unreadable", prefix, t.PrivateKey)
			}

		case "local":
			// No remote requirements, but RemotePath can still be used by actions; optional.
			// (Add local-specific requirements here if you introduce them later.)

		case "scp":
			// If you implement SCP later, mirror SFTP requirements as appropriate.
			// For now, treat like SFTP minus the SFTP client specifics:
			if strings.TrimSpace(t.Username) == "" {
				errs.addf("%s: username is required for SCP", prefix)
			}
			if strings.TrimSpace(t.Server) == "" {
				errs.addf("%s: server is required for SCP", prefix)
			}
			if !isValidPort(t.Port) {
				errs.addf("%s: port %q must be an integer 1-65535 for SCP", prefix, t.Port)
			}

		default:
			if strings.TrimSpace(t.TransferType) == "" {
				errs.addf("%s: transfertype is required", prefix)
			} else {
				errs.addf("%s: unsupported transfertype %q (allowed: sftp, local, scp)", prefix, t.TransferType)
			}
		}

		// Filter regex (if present)
		if strings.TrimSpace(t.Filter) != "" {
			if _, err := regexp.Compile(t.Filter); err != nil {
				errs.addf("%s: filter %q is not a valid regex: %v", prefix, t.Filter, err)
			}
		}

		// Success/Fail actions
		if !isValidAction(t.ActionOnSuccess) {
			errs.addf("%s: action_on_success %q invalid (allowed: none, archive, delete)", prefix, t.ActionOnSuccess)
		}
		if !isValidAction(t.ActionOnFail) {
			errs.addf("%s: action_on_fail %q invalid (allowed: none, archive, delete)", prefix, t.ActionOnFail)
		}

		// Archive/fail destinations if needed
		if isArchive(t.ActionOnSuccess) && strings.TrimSpace(t.ArchiveDest) != "" && !isDirOrCreatable(t.ArchiveDest) {
			errs.addf("%s: archive_dest %q does not exist and cannot be created", prefix, t.ArchiveDest)
		}
		if isArchive(t.ActionOnFail) && strings.TrimSpace(t.FailDest) != "" && !isDirOrCreatable(t.FailDest) {
			errs.addf("%s: fail_dest %q does not exist and cannot be created", prefix, t.FailDest)
		}
	}

	if errs.len() > 0 {
		return errs.err()
	}
	return nil
}

// ---- helpers ----

type multiErr struct {
	list []string
}

func (m *multiErr) addf(format string, a ...any) {
	m.list = append(m.list, fmt.Sprintf(format, a...))
}
func (m *multiErr) len() int { return len(m.list) }
func (m *multiErr) err() error {
	return errors.New(strings.Join(m.list, "; "))
}

func isValidLogLevel(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug", "info", "warn", "warning", "error", "":
		return true
	default:
		return false
	}
}

func isValidPort(p string) bool {
	if strings.TrimSpace(p) == "" {
		return false
	}
	n, err := strconv.Atoi(p)
	if err != nil || n < 1 || n > 65535 {
		return false
	}
	return true
}

func isValidAction(a string) bool {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "", "none", "archive", "delete":
		return true
	default:
		return false
	}
}
func isArchive(a string) bool {
	return strings.ToLower(strings.TrimSpace(a)) == "archive"
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
func isFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// isDirOrCreatable returns true if the path exists as a dir,
// or its parent exists and we can create it (we don't actually create it here).
func isDirOrCreatable(p string) bool {
	if isDir(p) {
		return true
	}
	parent := filepath.Dir(p)
	if !isDir(parent) {
		return false
	}
	// Check writability of parent by trying to open a file there (without creating directory p).
	f, err := os.CreateTemp(parent, ".permcheck-*")
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	return true
}
