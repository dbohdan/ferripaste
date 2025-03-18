package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/anmitsu/go-shlex"
	"github.com/alecthomas/repr"
	"github.com/otiai10/copy"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tdewolff/argp"
)

const (
	authzHeader      = "Authorization"
	dirPerms         = 0o700
	expireHeader     = "Expire"
	fieldFile        = "file"
	fieldOneshot     = "oneshot"
	fieldOneshotURL  = "oneshot_url"
	fieldRemote      = "remote"
	fieldURL         = "url"
	formatMaxBodyLen = 512
	timeout          = 30 * time.Second
)

// Config holds settings loaded from the config file.
type Config struct {
	URL          string `toml:"url"`
	Token        string `toml:"token"`
	TokenCommand string `toml:"token-command"`
}

// Upload represents a completed file upload.
type Upload struct {
	Name   string
	Status int
	URL    string
}

// Args holds parsed command-line arguments.
type Args struct {
	ConfigPath   string
	ExpireTime   string
	Filename     string
	Files        []string
	NoID         bool
	OneShot      bool
	RemoteURL    string
	StripExif    bool
	Suffix       string
	URLToShorten string
	Verbose      int
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		FormatLevel: func(i interface{}) string {
			return fmt.Sprintf("[%s]", i)
		},
		FormatMessage: func(i interface{}) string {
			return fmt.Sprintf("%s", i)
		},
		NoColor:      true,
		Out:          os.Stderr,
		PartsExclude: []string{"time"},
	})

	args := parseArgs()

	conf, err := loadConfig(args.ConfigPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	if err := run(args, conf); err != nil {
		if args.Verbose == 0 {
			// Just print the error message in normal mode.
			log.Fatal().Msg(err.Error())
		} else {
			// Print the full stack trace in verbose mode.
			log.Fatal().Err(err).Msg("Error during execution")
		}

		os.Exit(1)
	}
}

func parseArgs() Args {
	args := Args{
		Suffix: "",
	}

	p := argp.New("Ferripaste - Rustypaste client")

	p.AddOpt(&args.OneShot, "1", "one-shot", "One-shot upload")
	p.AddOpt(&args.ConfigPath, "c", "config", "Path to config file")
	p.AddOpt(&args.ExpireTime, "e", "expire", "Expiration time")
	p.AddOpt(&args.Filename, "f", "filename", "Custom filename")
	p.AddOpt(&args.NoID, "I", "no-id", "No Unix time ID suffix")
	p.AddOpt(&args.RemoteURL, "r", "remote", "Remote source URL")
	p.AddOpt(&args.StripExif, "s", "strip-exif", "Strip Exif metadata")
	p.AddOpt(&args.URLToShorten, "u", "url", "URL to shorten")
	p.AddOpt(argp.Count{&args.Verbose}, "v", "verbose", "Verbose mode")
	p.AddOpt(&args.Suffix, "x", "ext", "File suffix to add (including the \".\")")

	// Any remaining arguments are treated as files to upload.
	p.AddRest(&args.Files, "files", "Files to upload")

	p.Parse()

	if args.Verbose >= 2 {
		log.Debug().Msgf(
			"%s",
			repr.String(args, repr.Indent("    "), repr.OmitEmpty(false)),
		)
	}

	// Ensure exactly one upload source is specified.
	mutuallyExclusive := 0
	if len(args.Files) > 0 {
		mutuallyExclusive++
	}
	if args.RemoteURL != "" {
		mutuallyExclusive++
	}
	if args.URLToShorten != "" {
		mutuallyExclusive++
	}

	if mutuallyExclusive != 1 {
		log.Fatal().Msg("One or more file arguments, -r, or -u required")
	}

	if args.Filename != "" {
		if args.NoID {
			log.Fatal().Msg("Argument -I: not allowed with argument -f")
		}

		if len(args.Files) > 1 {
			log.Fatal().Msg("Argument -f: not allowed with more than one file")
		}
	}

	if args.OneShot && args.RemoteURL != "" {
		log.Fatal().Msg("Argument -1: not allowed with argument -r")
	}

	return args
}

func loadConfig(configPath string) (Config, error) {
	var config Config
	var configFile string

	if configPath != "" {
		configFile = configPath
	} else {
		var err error
		configFile, err = xdg.ConfigFile("ferripaste/config.toml")
		if err != nil {
			return Config{}, fmt.Errorf("failed to get config file path: %w", err)
		}
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := toml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Dynamically retrieve the token by executing "token-command" when "token" is not directly specified.
	if config.Token == "" && config.TokenCommand != "" {

		args, err := shlex.Split(config.TokenCommand, true)
		if err != nil {
			return Config{}, fmt.Errorf("failed to split token command: %v", err)
		}

		cmd := exec.Command(args[0], args[1:]...)
		output, err := cmd.Output()
		if err != nil {
			return Config{}, fmt.Errorf("failed to execute token command: %w", err)
		}

		config.Token = strings.TrimSpace(string(output))
	}

	return config, nil
}

func run(args Args, conf Config) error {
	var dests []string
	uploadID := strconv.FormatInt(time.Now().Unix(), 10)

	exitCode := 0

	if len(args.Files) > 0 {
		if args.Filename != "" {
			dests = []string{args.Filename}
		} else {
			for _, file := range args.Files {
				filename := filepath.Base(file)
				stem, ext := splitSuffix(filename)

				// Sanitize filename by replacing spaces and % with dashes.
				re := regexp.MustCompile(`[\s%]`)
				dest := re.ReplaceAllString(stem, "-")

				if !args.NoID {
					dest = dest + "." + uploadID
				}

				dest += ext + args.Suffix
				dests = append(dests, dest)
			}
		}
	}

	headers := map[string]string{
		authzHeader: conf.Token,
	}

	if args.ExpireTime != "" {
		headers[expireHeader] = args.ExpireTime
	}

	var field string
	var uploads []Upload
	var err error

	if args.RemoteURL != "" {
		field = fieldRemote
		uploads, err = uploadRemoteURL(conf.URL, headers, field, args.RemoteURL, args.Verbose)
	} else if args.URLToShorten != "" {
		if args.OneShot {
			field = fieldOneshotURL
		} else {
			field = fieldURL
		}
		uploads, err = uploadURL(conf.URL, headers, field, args.URLToShorten, args.Verbose)
	} else {
		if args.OneShot {
			field = fieldOneshot
		} else {
			field = fieldFile
		}

		// When requested, create copies of files with Exif data removed.
		processedFiles := args.Files
		var tempDir string

		if args.StripExif {
			tempDir, err = os.MkdirTemp("", "ferripaste-*")
			if err != nil {
				return fmt.Errorf("failed to create temp directory: %w", err)
			}
			defer os.RemoveAll(tempDir)

			processedFiles = make([]string, len(args.Files))
			for i, file := range args.Files {
				processedFile, err := copyWithoutExif(file, tempDir)
				if err != nil {
					return fmt.Errorf("failed to strip Exif data: %w", err)
				}
				processedFiles[i] = processedFile
			}
		}

		uploads, err = uploadFiles(conf.URL, headers, field, processedFiles, dests, args.Verbose)
	}

	if err != nil {
		return err
	}

	for _, upload := range uploads {
		if upload.URL != "" {
			fmt.Println(upload.URL)
		}
	}

	for _, upload := range uploads {
		if upload.Status != http.StatusOK {
			exitCode = 1
			break
		}
	}

	// Ensure uploaded files are accessible through their URLs.
	if !args.OneShot {
		if !verifyUploads(uploads) {
			exitCode = 1
		}
	}

	if exitCode != 0 {
		return errors.New("one or more uploads failed")
	}

	return nil
}

func uploadFiles(baseURL string, headers map[string]string, field string, files []string, dests []string, verbose int) ([]Upload, error) {
	client := &http.Client{
		Timeout: timeout,
	}
	uploads := make([]Upload, len(files))

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for i, file := range files {
		wg.Add(1)

		go func(i int, file, dest string) {
			defer wg.Done()

			upload, err := uploadFile(client, baseURL, headers, field, file, dest, verbose)

			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				uploads[i] = upload
			} else {
				errs = append(errs, err)
			}
		}(i, file, dests[i])
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return uploads, nil
}

func uploadFile(client *http.Client, baseURL string, headers map[string]string, field, file, dest string, verbose int) (Upload, error) {
	f, err := os.Open(file)
	if err != nil {
		return Upload{}, fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	writer := newMultipartWriter(&buf, field, dest, f)
	req, err := http.NewRequest("POST", baseURL, &buf)
	if err != nil {
		return Upload{}, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if verbose > 0 {
		logRequest(req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Upload{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Upload{}, fmt.Errorf("failed to read response: %w", err)
	}

	return Upload{
		Name:   filepath.Base(file),
		Status: resp.StatusCode,
		URL:    strings.TrimSpace(string(body)),
	}, nil
}

func uploadURL(baseURL string, headers map[string]string, field string, url string, verbose int) ([]Upload, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	var requestBody bytes.Buffer
	writer := newMultipartWriterForText(&requestBody, field, url)

	req, err := http.NewRequest("POST", baseURL, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if verbose > 0 {
		logRequest(req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return []Upload{
		{
			Name:   url,
			Status: resp.StatusCode,
			URL:    strings.TrimSpace(string(body)),
		},
	}, nil
}

func uploadRemoteURL(baseURL string, headers map[string]string, field string, url string, verbose int) ([]Upload, error) {
	return uploadURL(baseURL, headers, field, url, verbose)
}

func verifyUploads(uploads []Upload) bool {
	client := &http.Client{
		Timeout: timeout,
	}

	result := true

	for _, upload := range uploads {
		if upload.URL == "" {
			log.Error().Msgf("%s failed to upload with status %d", upload.Name, upload.Status)
			result = false
			continue
		}

		// Verify URL is accessible without downloading the full content.
		req, err := http.NewRequest("GET", upload.URL, nil)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to create verification request for %s", upload.URL)
			result = false
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			log.Error().Err(err).Msgf("Failed to verify URL %s", upload.URL)
			result = false
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Error().Msgf("%s failed URL verification with status %d", upload.URL, resp.StatusCode)
			result = false
		}

		resp.Body.Close()
		cancel()
	}

	return result
}

func copyWithoutExif(src, destDir string) (string, error) {
	i := 1
	destSubdir := destDir

	for {
		info, err := os.Stat(filepath.Join(destDir, strconv.Itoa(i)))

		if os.IsNotExist(err) {
			destSubdir = filepath.Join(destDir, strconv.Itoa(i))
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to check directory: %w", err)
		}

		if !info.IsDir() {
			return "", fmt.Errorf("%s is not a directory", filepath.Join(destDir, strconv.Itoa(i)))
		}

		i++
	}

	if err := os.Mkdir(destSubdir, dirPerms); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	dest := filepath.Join(destSubdir, filepath.Base(src))
	if err := copy.Copy(src, dest); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	cmd := exec.Command("exiftool", "-all=", "-quiet", dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to strip Exif metadata: %w", err)
	}

	return dest, nil
}

func splitSuffix(filename string) (string, string) {
	dotIndex := strings.Index(filename, ".")

	if dotIndex == -1 {
		return filename, ""
	}

	stem := filename[:dotIndex]
	ext := filename[dotIndex:] // Includes the leading period.

	return stem, ext
}

func logRequest(req *http.Request) {
	var headers []string

	for header, values := range req.Header {
		for _, value := range values {
			if header == authzHeader {
				value = "***"
			}

			headers = append(headers, fmt.Sprintf("%s: %s", header, value))
		}
	}

	log.Debug().Msgf("Request:\n    %s\n    %s", req.URL.String(), strings.Join(headers, "\n    "))
}

// newMultipartWriter creates a multipart writer for file uploads.
func newMultipartWriter(body *bytes.Buffer, fieldName, filename string, file *os.File) *multipart.Writer {
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile(fieldName, filename)
	io.Copy(part, file)
	writer.Close()
	return writer
}

// newMultipartWriterForText creates a multipart writer for text fields.
func newMultipartWriterForText(body *bytes.Buffer, fieldName, value string) *multipart.Writer {
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormField(fieldName)
	part.Write([]byte(value))
	writer.Close()
	return writer
}
