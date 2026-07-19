package config

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"sync"
)

var mu sync.Mutex

func AtomicWrite(path, content string) error {
	mu.Lock()
	defer mu.Unlock()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := []byte(path + ".tmp.")
	if _, err := rand.Read(tmp[len(path)+5:]); err != nil {
		return err
	}
	tmpPath := string(tmp)
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return err
	}
	if err := fileSync(tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

const (
	xdgPWConf = "pipewire/pipewire.conf.d/99-zen-dsp.conf"
	xdgZConf  = "zen-dsp/presets.json"
)

func XDGPipeWireConf() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, xdgPWConf)
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config", xdgPWConf)
}

func XDGZenDSP() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, xdgZConf)
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, ".config", xdgZConf)
}

func WritePreset(p []byte) error {
	return AtomicWrite(XDGZenDSP(), string(p))
}

func LoadPreset() ([]byte, error) {
	return os.ReadFile(XDGZenDSP())
}

func fileSync(name string) error {
	f, err := os.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}
