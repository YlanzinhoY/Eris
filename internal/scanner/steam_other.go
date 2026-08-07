//go:build !windows

package scanner

import (
	"os"
	"path/filepath"
)

func steamInstallRoots() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, "Library", "Application Support", "Steam"),
	}
}
