package services

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
)

func SolveCaptcha(conn *websocket.Conn, captchaMessage []interface{}) error {
	if len(captchaMessage) < 2 {
		return fmt.Errorf("invalid captcha message")
	}

	captchaData, ok := captchaMessage[1].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid captcha data format")
	}

	algorithm := toString(captchaData, "algorithm")
	challenge := toString(captchaData, "challenge")
	salt := toString(captchaData, "salt")
	signature := toString(captchaData, "signature")
	maxNumber := toIntFromAny(captchaData["maxnumber"])

	fmt.Printf("Solving captcha: algorithm=%s, challenge=%s, maxnumber=%d\n", algorithm, challenge, maxNumber)

	if maxNumber <= 0 {
		return fmt.Errorf("invalid maxnumber: %d", maxNumber)
	}

	startTime := time.Now()
	solution := solveChallengeSync(algorithm, challenge, salt, maxNumber)
	elapsed := int(time.Since(startTime).Milliseconds())

	solutionData := map[string]interface{}{
		"algorithm": algorithm,
		"challenge": challenge,
		"number":    solution,
		"salt":      salt,
		"signature": signature,
		"took":      elapsed,
	}

	solutionJSON, err := json.Marshal(solutionData)
	if err != nil {
		return fmt.Errorf("failed to marshal solution: %w", err)
	}

	solutionB64 := base64.StdEncoding.EncodeToString(solutionJSON)

	response := []interface{}{"cptR", solutionB64}
	packedResponse, err := msgpack.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to encode captcha response: %w", err)
	}

	err = conn.WriteMessage(websocket.BinaryMessage, append(packedResponse, 0x00, 0x00))
	if err != nil {
		return fmt.Errorf("failed to send captcha response: %w", err)
	}

	fmt.Printf("Solved captcha: %d (took %dms)\n", solution, elapsed)
	time.Sleep(1 * time.Second)

	return nil
}

func solveChallengeSync(algorithm, challenge, salt string, maxNumber int) int {
	target := challenge

	for i := 0; i <= maxNumber; i++ {
		data := salt + strconv.Itoa(i)

		var hash string
		if algorithm == "SHA-256" {
			h := sha256.Sum256([]byte(data))
			hash = fmt.Sprintf("%x", h)
		} else {
			continue
		}

		if hash == target {
			return i
		}
	}

	return 0
}
