// services/profile.go

package src

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
)

const wsURL = "wss://social.krunker.io/ws"

type PlayerStats struct {
	PlayerName      string `json:"player_name"`
	PlayerID        int    `json:"player_id"`
	PlayerScore     int    `json:"player_score"`
	PlayerFunds     int    `json:"player_funds"`
	PlayerSkinValue int    `json:"player_skinvalue"`
}

func FetchPlayerStats(username string, proxyURL string) (*PlayerStats, error) {
	if isEmail(username) {
		return nil, fmt.Errorf("cannot fetch stats for email addresses")
	}

	conn, err := createWebSocketConnection(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket connection: %w", err)
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	err = sendProfileRequest(conn, username)
	if err != nil {
		return nil, fmt.Errorf("failed to send profile request: %w", err)
	}

	stats, err := waitForProfileResponse(conn, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile response: %w", err)
	}

	return stats, nil
}

func GetUsernameFromWebSocket(account Account, proxyURL string) (string, error) {
	conn, err := createWebSocketConnection(proxyURL)
	if err != nil {
		return "", fmt.Errorf("failed to create websocket connection: %w", err)
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	token, err := getJWTToken(account, proxyURL)
	if err != nil {
		return "", fmt.Errorf("failed to get JWT token: %w", err)
	}

	if token == "" {
		return "", fmt.Errorf("empty JWT token")
	}

	err = sendWebSocketLogin(conn, token)
	if err != nil {
		return "", fmt.Errorf("failed to send websocket login: %w", err)
	}

	username, err := waitForLoginResponse(conn)
	if err != nil {
		return "", fmt.Errorf("failed to get username from login response: %w", err)
	}

	return username, nil
}

func CalculateLevel(score int) int {
	return int(math.Sqrt(float64(score) / 1111.0))
}

func createWebSocketConnection(proxyURL string) (*websocket.Conn, error) {
	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	dialer.HandshakeTimeout = 15 * time.Second

	if proxyURL != "" {
		if proxyURLParsed, err := url.Parse(proxyURL); err == nil {
			dialer.Proxy = http.ProxyURL(proxyURLParsed)
		}
	}

	headers := http.Header{}
	headers.Set("User-Agent", GenerateUserAgent())
	headers.Set("Origin", "https://krunker.io")

	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func sendProfileRequest(conn *websocket.Conn, username string) error {
	profileReq := []interface{}{"r", "profile", username}
	packedReq, err := msgpack.Marshal(profileReq)
	if err != nil {
		return fmt.Errorf("failed to encode profile request: %w", err)
	}

	return conn.WriteMessage(websocket.BinaryMessage, append(packedReq, 0x00, 0x00))
}

func sendWebSocketLogin(conn *websocket.Conn, token string) error {
	loginReq := []interface{}{"_0", 0, "login", token}
	packedReq, err := msgpack.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to encode login request: %w", err)
	}

	return conn.WriteMessage(websocket.BinaryMessage, append(packedReq, 0x00, 0x00))
}

func waitForProfileResponse(conn *websocket.Conn, username string) (*PlayerStats, error) {
	timeout := time.After(20 * time.Second)
	messageCount := 0

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for profile response")
		default:
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, message, err := conn.ReadMessage()
			if err != nil {
				if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
					continue
				}
				return nil, fmt.Errorf("failed to read websocket message: %w", err)
			}

			messageCount++

			if len(message) >= 2 {
				message = message[:len(message)-2]
			}

			var response []interface{}
			err = msgpack.Unmarshal(message, &response)
			if err != nil {
				continue
			}

			if len(response) >= 1 && response[0] == "cpt" {
				err = SolveCaptcha(conn, response)
				if err != nil {
					return nil, fmt.Errorf("failed to solve captcha: %w", err)
				}

				time.Sleep(2 * time.Second)

				err = sendProfileRequest(conn, username)
				if err != nil {
					return nil, fmt.Errorf("failed to resend profile request after captcha: %w", err)
				}
				continue
			}

			if len(response) >= 4 && response[3] != nil {
				if playerData, ok := response[3].(map[string]interface{}); ok {
					stats := parsePlayerStats(playerData)
					if stats != nil && stats.PlayerName != "" {
						return stats, nil
					}
				}
			}

			if messageCount > 15 {
				return nil, fmt.Errorf("profile data not found in response")
			}
		}
	}
}

func waitForLoginResponse(conn *websocket.Conn) (string, error) {
	timeout := time.After(20 * time.Second)
	messageCount := 0
	timeoutCount := 0
	maxTimeouts := 3

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for login response")
		default:
			conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) ||
					websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) ||
					strings.Contains(err.Error(), "repeated read on failed websocket connection") ||
					strings.Contains(err.Error(), "use of closed network connection") {
					return "", fmt.Errorf("websocket connection failed: %w", err)
				}

				if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
					timeoutCount++
					if timeoutCount >= maxTimeouts {
						return "", fmt.Errorf("connection timeout: too many consecutive read timeouts")
					}
					continue
				}

				return "", fmt.Errorf("failed to read websocket message: %w", err)
			}

			timeoutCount = 0
			messageCount++

			if len(message) >= 2 {
				message = message[:len(message)-2]
			}

			var response []interface{}
			err = msgpack.Unmarshal(message, &response)
			if err != nil {
				continue
			}

			if len(response) >= 1 && response[0] == "cpt" {
				err = SolveCaptcha(conn, response)
				if err != nil {
					return "", fmt.Errorf("failed to solve captcha: %w", err)
				}

				time.Sleep(2 * time.Second)

				continue
			}

			if len(response) >= 4 && response[0] == "a" {
				if username, ok := response[3].(string); ok && username != "" {
					return username, nil
				}
			}

			if messageCount > 15 {
				return "", fmt.Errorf("username not found in login response")
			}
		}
	}
}

func parsePlayerStats(playerData map[string]interface{}) *PlayerStats {
	playerName := mapx(playerData, "player_name")
	if playerName == "" {
		return nil
	}

	stats := &PlayerStats{
		PlayerName: playerName,
	}

	if id := toIntFromAny(playerData["player_id"]); id > 0 {
		stats.PlayerID = id
	}
	if scoreVal, exists := playerData["player_score"]; exists {
		stats.PlayerScore = toIntFromAny(scoreVal)
	}
	if fundsVal, exists := playerData["player_funds"]; exists {
		stats.PlayerFunds = toIntFromAny(fundsVal)
	}
	if skinVal, exists := playerData["player_skinvalue"]; exists {
		stats.PlayerSkinValue = toIntFromAny(skinVal)
	}

	if playerStatsStr := mapx(playerData, "player_stats"); playerStatsStr != "" {
		var playerStatsMap map[string]interface{}
		if err := json.Unmarshal([]byte(playerStatsStr), &playerStatsMap); err == nil {
		}
	}

	return stats
}

func mapx(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
