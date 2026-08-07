package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var steamLibraryPathPattern = regexp.MustCompile(`(?m)"path"\s*"([^"]+)"`)

func discoverSteamRoots() []root {
	installRoots := steamInstallRoots()
	libraryRoots := append([]string(nil), installRoots...)
	for _, installRoot := range installRoots {
		libraryRoots = append(libraryRoots, readSteamLibraries(installRoot)...)
	}

	candidates := make([]root, 0, len(libraryRoots)*2)
	for _, libraryRoot := range libraryRoots {
		if strings.TrimSpace(libraryRoot) == "" {
			continue
		}
		steamApps := filepath.Join(filepath.Clean(libraryRoot), "steamapps")
		candidates = append(candidates,
			root{path: filepath.Join(steamApps, "common"), source: "Steam"},
			root{path: filepath.Join(steamApps, "downloading"), source: "Steam"},
		)
	}
	return uniqueRoots(candidates)
}

func readSteamLibraries(installRoot string) []string {
	data, err := os.ReadFile(filepath.Join(installRoot, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		return nil
	}

	matches := steamLibraryPathPattern.FindAllStringSubmatch(string(data), -1)
	libraries := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		libraries = append(libraries, strings.ReplaceAll(match[1], `\\`, `\`))
	}
	return libraries
}
