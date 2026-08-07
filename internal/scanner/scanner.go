package scanner

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"
)

type Result struct {
	Path   string
	Source string
}

type Scanner struct {
	steamRoots    []root
	fallbackRoots []root
}

type root struct {
	path   string
	source string
}

type scanBatch struct {
	results []Result
	err     error
}

var ignoredDirectoryNames = map[string]struct{}{
	".git":         {},
	".hg":          {},
	".svn":         {},
	".venv":        {},
	"__pycache__":  {},
	"node_modules": {},
}

var ignoredRootDirectoryNames = map[string]struct{}{
	"$recycle.bin":              {},
	"$winreagent":               {},
	"config.msi":                {},
	"recovery":                  {},
	"system volume information": {},
	"windows":                   {},
}

func New(extraRoots []string) *Scanner {
	return &Scanner{
		steamRoots:    discoverSteamRoots(),
		fallbackRoots: discoverFallbackRoots(extraRoots),
	}
}

func (s *Scanner) Scan(ctx context.Context, gameName string, executableName string) ([]Result, error) {
	wanted := normalize(gameName)
	if wanted == "" {
		return nil, errors.New("informe o nome do jogo")
	}
	wantedExecutable := strings.TrimSpace(filepath.Base(executableName))

	results, err := scanRoots(ctx, s.steamRoots, wanted, wantedExecutable)
	if err != nil || len(results) > 0 {
		return results, err
	}
	return scanRoots(ctx, s.fallbackRoots, wanted, wantedExecutable)
}

func scanRoots(ctx context.Context, roots []root, wanted string, wantedExecutable string) ([]Result, error) {
	if len(roots) == 0 {
		return nil, nil
	}

	scanContext, stopScan := context.WithCancel(ctx)
	defer stopScan()
	batches := make(chan scanBatch, len(roots))
	var workers sync.WaitGroup
	for _, scanRoot := range roots {
		workers.Add(1)
		go func(scanRoot root) {
			defer workers.Done()
			found, err := walkRoot(scanContext, scanRoot, wanted, wantedExecutable)
			batches <- scanBatch{results: found, err: err}
		}(scanRoot)
	}
	go func() {
		workers.Wait()
		close(batches)
	}()

	results := make([]Result, 0, 1)
	seen := make(map[string]struct{})
	for batch := range batches {
		if batch.err != nil && errors.Is(batch.err, context.Canceled) && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		for _, result := range batch.results {
			addResult(&results, seen, result)
		}
		if len(batch.results) > 0 {
			stopScan()
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results, nil
}

func walkRoot(ctx context.Context, scanRoot root, wanted string, wantedExecutable string) ([]Result, error) {
	if _, err := os.Stat(scanRoot.path); err != nil {
		return nil, nil
	}

	results := make([]Result, 0, 1)
	seen := make(map[string]struct{})
	err := filepath.WalkDir(scanRoot.path, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if entry.IsDir() {
			if entry.Type()&os.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			if shouldIgnoreDirectory(entry.Name(), relativeDepth(scanRoot.path, path)) {
				return filepath.SkipDir
			}
			if wantedExecutable == "" && matchesGame(normalize(entry.Name()), wanted) {
				addResult(&results, seen, Result{Path: path, Source: scanRoot.source})
				return filepath.SkipAll
			}
			return nil
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if wantedExecutable != "" {
			if strings.EqualFold(entry.Name(), wantedExecutable) {
				addResult(&results, seen, Result{Path: filepath.Dir(path), Source: scanRoot.source})
				return filepath.SkipAll
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".exe") {
			return nil
		}
		executableName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if matchesGame(normalize(executableName), wanted) {
			addResult(&results, seen, Result{Path: filepath.Dir(path), Source: scanRoot.source})
			return filepath.SkipAll
		}
		return nil
	})
	if errors.Is(err, context.Canceled) {
		return nil, err
	}
	return results, nil
}

func discoverFallbackRoots(extraRoots []string) []root {
	candidates := make([]root, 0, len(extraRoots)+4)
	for _, path := range extraRoots {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		candidates = append(candidates, root{path: filepath.Clean(trimmed), source: "Prioritária"})
	}
	candidates = append(candidates, localDriveRoots()...)
	return uniqueRoots(candidates)
}

func uniqueRoots(candidates []root) []root {
	unique := make([]root, 0, len(candidates))
	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.path) == "" {
			continue
		}
		candidate.path = filepath.Clean(candidate.path)
		key := strings.ToLower(candidate.path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func shouldIgnoreDirectory(name string, depth int) bool {
	name = strings.ToLower(name)
	if _, ignored := ignoredDirectoryNames[name]; ignored {
		return true
	}
	if depth == 1 {
		_, ignored := ignoredRootDirectoryNames[name]
		return ignored
	}
	return false
}

func relativeDepth(rootPath string, path string) int {
	relative, err := filepath.Rel(rootPath, path)
	if err != nil || relative == "." {
		return 0
	}
	return strings.Count(relative, string(os.PathSeparator)) + 1
}

func addResult(results *[]Result, seen map[string]struct{}, result Result) {
	if strings.TrimSpace(result.Path) == "" {
		return
	}
	key := strings.ToLower(filepath.Clean(result.Path))
	if _, exists := seen[key]; exists {
		return
	}
	seen[key] = struct{}{}
	*results = append(*results, result)
}

func matchesGame(candidate string, wanted string) bool {
	if candidate == "" || wanted == "" {
		return false
	}
	if candidate == wanted {
		return true
	}
	return len(wanted) >= 4 && strings.Contains(candidate, wanted)
}

func normalize(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}
