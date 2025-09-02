// services/proxy.go

package src

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type ProxyManager struct {
	activeProxies     []string
	shelvedProxies    map[string]time.Time
	removedProxies    map[string]bool
	currentIndex      int
	mu                sync.Mutex
	initialProxies    []string
	initialProxyCount int
	lastRestore       time.Time
}

var proxyManager *ProxyManager

const (
	shelveTimeout   = 2 * time.Minute
	restoreInterval = 30 * time.Second
)

func InitializeProxyManager() error {
	proxies, err := readem("data/proxies.txt")
	if err != nil {
		return fmt.Errorf("failed to read proxies.txt: %w", err)
	}
	if len(proxies) == 0 {
		return fmt.Errorf("no valid proxies found in proxies.txt")
	}

	proxyManager = &ProxyManager{
		activeProxies:     make([]string, len(proxies)),
		shelvedProxies:    make(map[string]time.Time),
		removedProxies:    make(map[string]bool),
		currentIndex:      -1,
		initialProxies:    make([]string, len(proxies)),
		initialProxyCount: len(proxies),
		lastRestore:       time.Now(),
	}

	copy(proxyManager.activeProxies, proxies)
	copy(proxyManager.initialProxies, proxies)

	return nil
}

func (pm *ProxyManager) GetNextProxy() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if time.Since(pm.lastRestore) > restoreInterval {
		pm.restoreShelvedProxiesInternal()
		pm.lastRestore = time.Now()
	}

	if len(pm.activeProxies) == 0 {
		pm.restoreAllProxiesInternal()
	}

	if len(pm.activeProxies) == 0 {
		return ""
	}

	pm.currentIndex = (pm.currentIndex + 1) % len(pm.activeProxies)
	return pm.activeProxies[pm.currentIndex]
}

func (pm *ProxyManager) RemoveProxy(proxyToRemove string) {
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

func (pm *ProxyManager) ShelveProxy(proxyToShelve string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.removedProxies[proxyToShelve] {
		return
	}

	if _, exists := pm.shelvedProxies[proxyToShelve]; exists {
		return
	}

	newActiveProxies := []string{}
	for _, p := range pm.activeProxies {
		if p != proxyToShelve {
			newActiveProxies = append(newActiveProxies, p)
		}
	}

	pm.activeProxies = newActiveProxies
	pm.shelvedProxies[proxyToShelve] = time.Now()
	pm.currentIndex = -1
}

func (pm *ProxyManager) restoreShelvedProxiesInternal() {
	now := time.Now()
	var restoredCount int

	for proxy, shelveTime := range pm.shelvedProxies {
		if now.Sub(shelveTime) > shelveTimeout {
			pm.activeProxies = append(pm.activeProxies, proxy)
			delete(pm.shelvedProxies, proxy)
			restoredCount++
		}
	}

	if restoredCount > 0 {
		pm.currentIndex = -1
	}
}

func (pm *ProxyManager) restoreAllProxiesInternal() {
	var restoredCount int

	for proxy := range pm.shelvedProxies {
		pm.activeProxies = append(pm.activeProxies, proxy)
		delete(pm.shelvedProxies, proxy)
		restoredCount++
	}

	if len(pm.activeProxies) == 0 {
		for _, proxy := range pm.initialProxies {
			if !pm.removedProxies[proxy] {
				pm.activeProxies = append(pm.activeProxies, proxy)
				restoredCount++
			}
		}
	}

	if restoredCount > 0 {
		pm.currentIndex = -1
	}
}

func (pm *ProxyManager) GetProxyStats() (active int, bad int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	active = len(pm.activeProxies)
	shelved := len(pm.shelvedProxies)
	removed := len(pm.removedProxies)
	bad = shelved + removed
	return
}

func (pm *ProxyManager) SaveGoodProxies() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	goodProxies := make(map[string]bool)

	for _, p := range pm.activeProxies {
		goodProxies[p] = true
	}

	for p := range pm.shelvedProxies {
		goodProxies[p] = true
	}

	var finalProxyList []string
	for p := range goodProxies {
		cleanProxy := p
		if strings.Contains(p, "@") {
			finalProxyList = append(finalProxyList, cleanProxy)
		} else {
			cleanProxy = strings.TrimPrefix(cleanProxy, "http://")
			cleanProxy = strings.TrimPrefix(cleanProxy, "https://")
			cleanProxy = strings.TrimPrefix(cleanProxy, "socks5://")
			finalProxyList = append(finalProxyList, cleanProxy)
		}
	}

	content := strings.Join(finalProxyList, "\n")
	err := os.WriteFile("data/proxies.txt", []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("could not write cleaned proxies file: %w", err)
	}

	return nil
}

func hh1(proxyURL string) *http.Client {
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

func validate(proxyStr string) bool {
	if proxyStr == "" {
		return false
	}

	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return false
	}

	if proxyURL.Scheme != "http" && proxyURL.Scheme != "https" && proxyURL.Scheme != "socks5" {
		return false
	}

	if proxyURL.Host == "" {
		return false
	}

	return true
}

func readem(filename string) ([]string, error) {
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

		normalizedProxy := normalize(line)
		if normalizedProxy != "" && validate(normalizedProxy) {
			proxies = append(proxies, normalizedProxy)
		}
	}

	return proxies, scanner.Err()
}

func normalize(proxy string) string {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return ""
	}

	if strings.HasPrefix(proxy, "http://") ||
		strings.HasPrefix(proxy, "https://") ||
		strings.HasPrefix(proxy, "socks5://") {
		return proxy
	}

	if strings.Contains(proxy, "@") {
		if _, err := url.Parse("http://" + proxy); err == nil {
			return "http://" + proxy
		}
	}

	return "http://" + proxy
}

func (pm *ProxyManager) GetActiveProxyCount() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.activeProxies)
}

func (pm *ProxyManager) GetTotalProxyCount() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.initialProxyCount
}

func (pm *ProxyManager) RestoreShelvedProxies() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.restoreShelvedProxiesInternal()
}

func proxym() *ProxyManager {
	return proxyManager
}
