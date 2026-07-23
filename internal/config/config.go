package config

import (
	"fmt"
	"os"
)

// Config contains runtime directory and file paths for whatsd.
type Config struct {
	DataDir    string
	StateDir   string
	RuntimeDir string
	DBPath     string
	SocketPath string
}

// LoadConfig resolves XDG compliant directory paths for whatsd.
func LoadConfig() Config {
	home, _ := os.UserHomeDir()

	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = home + "/.local/share"
	}
	dataDir := dataHome + "/whatsd"

	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = home + "/.local/state"
	}
	stateDir := stateHome + "/whatsd"

	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = stateDir
	} else {
		runtimeDir = runtimeDir + "/whatsd"
	}

	return Config{
		DataDir:    dataDir,
		StateDir:   stateDir,
		RuntimeDir: runtimeDir,
		DBPath:     dataDir + "/session.db",
		SocketPath: runtimeDir + "/whatsd.sock",
	}
}

// EnsureDirectories creates required data, state, and runtime directories.
func (c *Config) EnsureDirectories() error {
	for _, dir := range []string{c.DataDir, c.StateDir, c.RuntimeDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}
