package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

const (
	progressInterval = 100 * time.Millisecond
	copyBufferSize   = 256 * 1024
)

type Progress struct {
	Downloaded int64
	Total      int64
}

type Result struct {
	Path  string
	Bytes int64
}

var httpClient = newHTTPClient()

func Download(
	ctx context.Context,
	rawURL string,
	destinationDirectory string,
	onProgress func(Progress),
) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return Result{}, errors.New("link de download inválido")
	}
	if parsedURL.User != nil {
		return Result{}, errors.New("link de download não pode conter credenciais")
	}

	destinationDirectory, err = filepath.Abs(destinationDirectory)
	if err != nil {
		return Result{}, fmt.Errorf("resolver pasta de destino: %w", err)
	}
	destinationInfo, err := os.Stat(destinationDirectory)
	if err != nil {
		return Result{}, fmt.Errorf("acessar pasta do jogo: %w", err)
	}
	if !destinationInfo.IsDir() {
		return Result{}, fmt.Errorf("destino não é uma pasta: %s", destinationDirectory)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return Result{}, fmt.Errorf("criar requisição: %w", err)
	}
	request.Header.Set("User-Agent", "hv-game-cli/1.0")
	request.Header.Set("Accept", "application/octet-stream, application/zip, application/x-rar-compressed, */*")

	response, err := httpClient.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("iniciar download: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return Result{}, fmt.Errorf("servidor respondeu HTTP %d", response.StatusCode)
	}

	mediaType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if strings.EqualFold(mediaType, "text/html") {
		return Result{}, errors.New("o link retornou uma página HTML, não um arquivo")
	}
	fileName := responseFileName(response, mediaType)
	finalPath, err := availablePath(destinationDirectory, fileName)
	if err != nil {
		return Result{}, err
	}

	temporary, err := os.CreateTemp(destinationDirectory, ".hv-game-cli-*.part")
	if err != nil {
		return Result{}, fmt.Errorf("criar arquivo temporário na pasta do jogo: %w", err)
	}
	temporaryPath := temporary.Name()
	completed := false
	defer func() {
		_ = temporary.Close()
		if !completed {
			_ = os.Remove(temporaryPath)
		}
	}()

	total := response.ContentLength
	if total < 0 {
		total = 0
	}
	report(onProgress, Progress{Total: total})

	buffer := make([]byte, copyBufferSize)
	var downloaded int64
	lastReport := time.Now()
	for {
		bytesRead, readErr := response.Body.Read(buffer)
		if bytesRead > 0 {
			bytesWritten, writeErr := temporary.Write(buffer[:bytesRead])
			if writeErr != nil {
				return Result{}, fmt.Errorf("gravar download: %w", writeErr)
			}
			if bytesWritten != bytesRead {
				return Result{}, io.ErrShortWrite
			}
			downloaded += int64(bytesWritten)
		}

		if time.Since(lastReport) >= progressInterval || errors.Is(readErr, io.EOF) {
			report(onProgress, Progress{Downloaded: downloaded, Total: total})
			lastReport = time.Now()
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return Result{}, fmt.Errorf("receber download: %w", readErr)
		}
	}

	if err := temporary.Sync(); err != nil {
		return Result{}, fmt.Errorf("sincronizar download: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return Result{}, fmt.Errorf("fechar download: %w", err)
	}
	finalPath, err = availablePath(destinationDirectory, filepath.Base(finalPath))
	if err != nil {
		return Result{}, err
	}
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		return Result{}, fmt.Errorf("finalizar download: %w", err)
	}
	completed = true
	return Result{Path: finalPath, Bytes: downloaded}, nil
}

func newHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 30 * time.Second
	transport.TLSHandshakeTimeout = 15 * time.Second
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(request *http.Request, previous []*http.Request) error {
			if len(previous) >= 10 {
				return errors.New("muitos redirecionamentos no download")
			}
			if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
				return errors.New("redirecionamento para protocolo não permitido")
			}
			return nil
		},
	}
}

func responseFileName(response *http.Response, mediaType string) string {
	if _, parameters, err := mime.ParseMediaType(response.Header.Get("Content-Disposition")); err == nil {
		if fileName := sanitizeFileName(parameters["filename"]); fileName != "" {
			return fileName
		}
	}
	if decodedPath, err := url.PathUnescape(response.Request.URL.Path); err == nil {
		if fileName := sanitizeFileName(filepath.Base(decodedPath)); fileName != "" {
			if filepath.Ext(fileName) == "" {
				fileName += firstExtension(mediaType)
			}
			return fileName
		}
	}
	return "download" + firstExtension(mediaType)
}

func firstExtension(mediaType string) string {
	if mediaType == "" || strings.EqualFold(mediaType, "application/octet-stream") {
		return ".bin"
	}
	extensions, err := mime.ExtensionsByType(mediaType)
	if err == nil && len(extensions) > 0 {
		return extensions[0]
	}
	return ".bin"
}

func sanitizeFileName(value string) string {
	value = strings.TrimSpace(filepath.Base(value))
	value = strings.Map(func(character rune) rune {
		if character < 32 || unicode.IsControl(character) || strings.ContainsRune(`<>:"/\|?*`, character) {
			return '_'
		}
		return character
	}, value)
	value = strings.Trim(value, " .")
	if value == "" || value == "." || value == ".." {
		return ""
	}
	runes := []rune(value)
	if len(runes) > 180 {
		value = string(runes[:180])
	}
	baseName := strings.ToUpper(strings.TrimSuffix(value, filepath.Ext(value)))
	if isReservedWindowsName(baseName) {
		value = "_" + value
	}
	return value
}

func isReservedWindowsName(value string) bool {
	if value == "CON" || value == "PRN" || value == "AUX" || value == "NUL" {
		return true
	}
	if len(value) == 4 && (strings.HasPrefix(value, "COM") || strings.HasPrefix(value, "LPT")) {
		return value[3] >= '1' && value[3] <= '9'
	}
	return false
}

func availablePath(directory string, fileName string) (string, error) {
	fileName = sanitizeFileName(fileName)
	if fileName == "" {
		fileName = "download.bin"
	}
	for suffix := 0; suffix < 10_000; suffix++ {
		candidateName := fileName
		if suffix > 0 {
			extension := filepath.Ext(fileName)
			candidateName = fmt.Sprintf("%s (%d)%s", strings.TrimSuffix(fileName, extension), suffix, extension)
		}
		candidate := filepath.Join(directory, candidateName)
		relative, err := filepath.Rel(directory, candidate)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
			return "", errors.New("nome de arquivo de download inválido")
		}
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("verificar destino do download: %w", err)
		}
	}
	return "", errors.New("não foi possível escolher um nome livre para o download")
}

func report(callback func(Progress), progress Progress) {
	if callback != nil {
		callback(progress)
	}
}
