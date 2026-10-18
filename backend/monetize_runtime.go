package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/url"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var monetizeDataMu sync.Mutex
var monetizeSafeName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,119}$`)

func monetizeDataPath(name string) (string, error) {
	if !monetizeSafeName.MatchString(name) || strings.Contains(name, "..") {
		return "", errors.New("invalid data name")
	}
	root := os.Getenv("MONETIZEAI_DATA_DIR")
	if root == "" {
		root = "./data"
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

// monetizeLoadJSON and monetizeSaveJSON provide durable project storage. Data
// survives code changes, container restarts and new deployments.
func monetizeLoadJSON(name string, target any) error {
	monetizeDataMu.Lock()
	defer monetizeDataMu.Unlock()
	path, err := monetizeDataPath(name)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func monetizeSaveJSON(name string, value any) error {
	monetizeDataMu.Lock()
	defer monetizeDataMu.Unlock()
	path, err := monetizeDataPath(name)
	if err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil || len(data) > 8<<20 {
		if err == nil {
			err = errors.New("data document exceeds 8 MB")
		}
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, data, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func monetizeNewToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func monetizePBKDF2(password string, salt []byte, rounds int) []byte {
	mac := hmac.New(sha256.New, []byte(password))
	mac.Write(salt)
	mac.Write([]byte{0, 0, 0, 1})
	result := mac.Sum(nil)
	block := append([]byte(nil), result...)
	for index := 1; index < rounds; index++ {
		mac = hmac.New(sha256.New, []byte(password))
		mac.Write(block)
		block = mac.Sum(nil)
		for offset := range result {
			result[offset] ^= block[offset]
		}
	}
	return result
}

func monetizeHashPassword(password string) (string, error) {
	if len(password) < 8 || len(password) > 256 {
		return "", errors.New("password must contain 8 to 256 characters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return "pbkdf2-sha256$210000$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(monetizePBKDF2(password, salt, 210000)), nil
}

func monetizeVerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" || parts[1] != "210000" {
		return false
	}
	salt, saltErr := hex.DecodeString(parts[2])
	expected, hashErr := hex.DecodeString(parts[3])
	if saltErr != nil || hashErr != nil || len(salt) != 16 || len(expected) != 32 {
		return false
	}
	actual := monetizePBKDF2(password, salt, 210000)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func monetizeSaveUpload(header *multipart.FileHeader) (string, error) {
	if header == nil || header.Size <= 0 || header.Size > 8<<20 {
		return "", errors.New("upload must be between 1 byte and 8 MB")
	}
	extension := strings.ToLower(filepath.Ext(filepath.Base(header.Filename)))
	if len(extension) > 12 || !regexp.MustCompile(`^\.[a-z0-9]+$`).MatchString(extension) {
		extension = ".bin"
	}
	token, err := monetizeNewToken()
	if err != nil {
		return "", err
	}
	name := "upload-" + token + extension
	path, err := monetizeDataPath(name)
	if err != nil {
		return "", err
	}
	source, err := header.Open()
	if err != nil {
		return "", err
	}
	defer source.Close()
	target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(target, io.LimitReader(source, (8<<20)+1))
	closeErr := target.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		if copyErr != nil {
			return "", copyErr
		}
		return "", closeErr
	}
	return name, nil
}

type monetizeExternalResponse struct {
	Status      int    `json:"status"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
	Error       string `json:"error"`
}

// monetizeExternalJSON calls an HTTPS API through the platform's allowlisted
// gateway. secretName is optional; the secret itself is never exposed to code.
func monetizeExternalJSON(ctx context.Context, method, targetURL string, payload any, secretName, secretHeader string, output any) (int, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, err
		}
	}
	requestBody, err := json.Marshal(map[string]any{"url": targetURL, "method": method, "headers": map[string]string{"Accept": "application/json", "Content-Type": "application/json"}, "body": base64.StdEncoding.EncodeToString(body), "secret_name": secretName, "secret_header": secretHeader})
	if err != nil {
		return 0, err
	}
	if os.Getenv("MONETIZEAI_SELF_HOSTED") == "1" {
		return monetizeDirectExternalJSON(ctx, method, targetURL, body, secretName, secretHeader, output)
	}
	socket := os.Getenv("MONETIZEAI_EGRESS_SOCKET")
	token := os.Getenv("MONETIZEAI_EGRESS_TOKEN")
	if socket == "" || token == "" {
		return 0, errors.New("external gateway is not configured")
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	client := &http.Client{Transport: transport}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://monetizeai/v1/fetch", bytes.NewReader(requestBody))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-MonetizeAI-Egress-Token", token)
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	var result monetizeExternalResponse
	if json.NewDecoder(io.LimitReader(response.Body, 6<<20)).Decode(&result) != nil || result.Error != "" {
		if result.Error == "" {
			result.Error = "external request failed"
		}
		return result.Status, errors.New(result.Error)
	}
	decoded, err := base64.StdEncoding.DecodeString(result.Body)
	if err != nil {
		return result.Status, err
	}
	if output != nil && len(decoded) > 0 {
		if err := json.Unmarshal(decoded, output); err != nil {
			return result.Status, err
		}
	}
	return result.Status, nil
}

// monetizeDirectExternalJSON is used when the project runs on the owner's own
// server. The allowlist and secrets come from the service environment with the
// same rules as the platform gateway.
func monetizeDirectExternalJSON(ctx context.Context, method, targetURL string, body []byte, secretName, secretHeader string, output any) (int, error) {
	target, err := url.Parse(targetURL)
	if err != nil || target.Scheme != "https" || target.User != nil || target.Port() != "" {
		return 0, errors.New("host is not allowed")
	}
	host := strings.ToLower(strings.TrimSuffix(target.Hostname(), "."))
	allowed := false
	for _, item := range strings.Split(os.Getenv("MONETIZEAI_EGRESS_HOSTS"), ",") {
		if item = strings.TrimSpace(item); item != "" && strings.EqualFold(item, host) {
			allowed = true
		}
	}
	if !allowed {
		return 0, errors.New("host is not allowed")
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}
	request, err := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	if secretName != "" {
		secret := os.Getenv("MONETIZEAI_SECRET_" + strings.ToUpper(strings.TrimSpace(secretName)))
		header := http.CanonicalHeaderKey(secretHeader)
		if header == "" {
			header = "Authorization"
		}
		if secret == "" || (header != "Authorization" && header != "X-Api-Key" && header != "Api-Key") {
			return 0, errors.New("secret is not configured")
		}
		if header == "Authorization" {
			secret = "Bearer " + secret
		}
		request.Header.Set(header, secret)
	}
	client := &http.Client{Timeout: 18 * 1e9, CheckRedirect: func(next *http.Request, via []*http.Request) error {
		if len(via) >= 3 || !strings.EqualFold(next.URL.Hostname(), host) || next.URL.Scheme != "https" {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return response.StatusCode, err
	}
	if output != nil && len(data) > 0 {
		if err := json.Unmarshal(data, output); err != nil {
			return response.StatusCode, err
		}
	}
	return response.StatusCode, nil
}
