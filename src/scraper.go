// services/scraper.go

package src

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// add more sources if needed - github raw sources only
var proxySources = []string{
	"https://raw.githubusercontent.com/yemixzy/proxy-list/refs/heads/main/proxies/http.txt",
	"https://raw.githubusercontent.com/yemixzy/proxy-list/refs/heads/main/proxies/socks5.txt",
	"https://raw.githubusercontent.com/claude89757/free_https_proxies/refs/heads/main/https_proxies.txt",
	"https://raw.githubusercontent.com/Eloco/free-proxy-raw/refs/heads/main/proxy/http_1756099411_iw4p_C115.txt",
	"https://raw.githubusercontent.com/Eloco/free-proxy-raw/refs/heads/main/proxy/http_1756107704_iw4p_C106.txt",
	"https://raw.githubusercontent.com/Eloco/free-proxy-raw/refs/heads/main/proxy/http_1756106558_iw4p_C104.txt",
}

type ProxyScraper struct {
	sources []string
	timeout time.Duration
}

func proxys() *ProxyScraper {
	return &ProxyScraper{
		sources: proxySources,
		timeout: 15 * time.Second,
	}
}

func (ps *ProxyScraper) scproxies() ([]string, error) {
	fmt.Printf(ColorCyan+"Scraping proxies from %d sources...\n"+ColorReset, len(ps.sources))

	var allProxies []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	progressChan := make(chan string, len(ps.sources))
	go ps.displayProgress(progressChan, len(ps.sources))
	for i, source := range ps.sources {
		wg.Add(1)
		go func(url string, index int) {
			defer wg.Done()

			proxies, err := ps.sfromsource(url)
			sourceName := fmt.Sprintf("Source %d", index+1)

			mu.Lock()
			if err != nil {
				progressChan <- fmt.Sprintf("%s - Error: %v", sourceName, err)
			} else {
				allProxies = append(allProxies, proxies...)
				progressChan <- fmt.Sprintf("%s - Found %d proxies", sourceName, len(proxies))
			}
			mu.Unlock()
		}(source, i)
	}

	wg.Wait()
	close(progressChan)

	uniqueProxies := ps.removeDuplicates(allProxies)
	cleanedProxies := ps.cleanProxies(uniqueProxies)

	fmt.Printf(ColorGreen + "\nScraping completed!\n" + ColorReset)
	fmt.Printf(ColorCyan+"Total proxies found: %s%d%s\n"+ColorReset,
		ColorGreen, len(cleanedProxies), ColorReset)

	return cleanedProxies, nil
}

func (ps *ProxyScraper) sfromsource(url string) ([]string, error) {
	client := &http.Client{
		Timeout: ps.timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return ps.parseProxies(string(body)), nil
}

func (ps *ProxyScraper) parseProxies(content string) []string {
	var proxies []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") ||
			strings.Contains(line, "=") || strings.Contains(line, "://github.com") ||
			len(line) < 7 {
			continue
		}

		if ps.formatp(line) {
			proxies = append(proxies, line)
		}
	}

	return proxies
}

func (ps *ProxyScraper) formatp(proxy string) bool {
	cleanProxy := proxy
	if strings.HasPrefix(proxy, "http://") {
		cleanProxy = strings.TrimPrefix(proxy, "http://")
	} else if strings.HasPrefix(proxy, "https://") {
		cleanProxy = strings.TrimPrefix(proxy, "https://")
	} else if strings.HasPrefix(proxy, "socks5://") {
		cleanProxy = strings.TrimPrefix(proxy, "socks5://")
	}

	parts := strings.Split(cleanProxy, ":")
	if len(parts) != 2 {
		return false
	}

	if len(parts[1]) < 1 || len(parts[1]) > 5 {
		return false
	}

	if len(parts[0]) < 3 || len(parts[0]) > 255 {
		return false
	}

	return true
}
func (ps *ProxyScraper) removeDuplicates(proxies []string) []string {
	seen := make(map[string]bool)
	var unique []string

	for _, proxy := range proxies {
		normalized := strings.TrimPrefix(proxy, "http://")
		normalized = strings.TrimPrefix(normalized, "https://")
		normalized = strings.TrimPrefix(normalized, "socks5://")

		if !seen[normalized] {
			seen[normalized] = true
			unique = append(unique, proxy)
		}
	}

	return unique
}

func (ps *ProxyScraper) cleanProxies(proxies []string) []string {
	var cleaned []string
	for _, proxy := range proxies {
		if !strings.HasPrefix(proxy, "http://") &&
			!strings.HasPrefix(proxy, "https://") &&
			!strings.HasPrefix(proxy, "socks5://") {
			proxy = "http://" + proxy
		}

		cleaned = append(cleaned, proxy)
	}

	return cleaned
}

func (ps *ProxyScraper) displayProgress(progressChan <-chan string, total int) {
	completed := 0
	for message := range progressChan {
		completed++
		fmt.Printf("[%d/%d] %s\n", completed, total, message)
	}
}

func (ps *ProxyScraper) SaveProxiesToFile(proxies []string, filename string) error {
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create proxies file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString("# Auto-scraped proxies - " + time.Now().Format("2006-01-02 15:04:05") + "\n")
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for _, proxy := range proxies {
		cleanProxy := strings.TrimPrefix(proxy, "http://")
		cleanProxy = strings.TrimPrefix(cleanProxy, "https://")
		cleanProxy = strings.TrimPrefix(cleanProxy, "socks5://")

		_, err = file.WriteString(cleanProxy + "\n")
		if err != nil {
			return fmt.Errorf("failed to write proxy: %w", err)
		}
	}

	fmt.Printf(ColorGreen+"Saved %d proxies to %s\n"+ColorReset, len(proxies), filename)
	return nil
}
func PromptForProxyScraping() bool {
	fmt.Printf(ColorCyan + "scrape proxies? (y/N): " + ColorReset)

	var input string
	fmt.Scanln(&input)

	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes"
}
func ScrapeAndSaveProxies() error {
	scraper := proxys()
	proxies, err := scraper.scproxies()
	if err != nil {
		return fmt.Errorf("failed to scrape proxies: %w", err)
	}
	if len(proxies) == 0 {
		return fmt.Errorf("no proxies were scraped from any source")
	}
	err = scraper.SaveProxiesToFile(proxies, "data/proxies.txt")
	if err != nil {
		return fmt.Errorf("failed to save proxies: %w", err)
	}

	return nil
}
