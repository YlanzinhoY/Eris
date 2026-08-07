package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type Game struct {
	Name         string `json:"game"`
	DownloadLink string `json:"download_link"`
	Version      string `json:"version"`
	Executable   string `json:"exe"`
}

func Load(path string) ([]Game, error) {
	resolved, err := resolvePath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("ler catálogo %s: %w", resolved, err)
	}

	var games []Game
	if err := json.Unmarshal(data, &games); err != nil {
		return nil, fmt.Errorf("decodificar catálogo: %w", err)
	}
	if len(games) == 0 {
		return nil, errors.New("o catálogo não contém jogos")
	}

	for index, game := range games {
		if strings.TrimSpace(game.Name) == "" || strings.TrimSpace(game.Version) == "" {
			return nil, fmt.Errorf("jogo %d possui nome ou versão vazios", index+1)
		}
		parsedURL, parseErr := url.ParseRequestURI(game.DownloadLink)
		if parseErr != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return nil, fmt.Errorf("jogo %q possui download_link inválido", game.Name)
		}
		if executable := strings.TrimSpace(game.Executable); executable != "" && filepath.Base(executable) != executable {
			return nil, fmt.Errorf("jogo %q possui exe inválido: informe apenas o nome do arquivo", game.Name)
		}
	}

	return games, nil
}

func Find(games []Game, query string) (Game, error) {
	wanted := normalize(query)
	if wanted == "" {
		return Game{}, errors.New("informe um nome de jogo válido")
	}
	for _, game := range games {
		if normalize(game.Name) == wanted {
			return game, nil
		}
	}
	for _, game := range games {
		if strings.Contains(normalize(game.Name), wanted) {
			return game, nil
		}
	}
	return Game{}, fmt.Errorf("jogo %q não encontrado no catálogo", query)
}

func resolvePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	executable, err := os.Executable()
	if err != nil {
		return path, nil
	}
	candidate := filepath.Join(filepath.Dir(executable), path)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("catálogo %q não encontrado", path)
}

func normalize(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}
