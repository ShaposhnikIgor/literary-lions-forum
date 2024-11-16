package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

type Captcha struct {
	Question  string
	Answer    string
	ExpiresAt time.Time
}

func GenerateCaptcha() Captcha {
	rand.Seed(time.Now().UnixNano())
	var a, b, result int
	var op string

	for {
		a = rand.Intn(10)
		b = rand.Intn(10)
		switch rand.Intn(4) {
		case 0:
			op = "+"
			result = a + b
		case 1:
			op = "-"
			result = a - b
		case 2:
			op = "*"
			result = a * b
		case 3:
			if b != 0 && a%b == 0 {
				op = "/"
				result = a / b
			} else {
				continue
			}
		}
		if result >= 0 && result <= 9 {
			break
		}
	}

	question := fmt.Sprintf("%d %s %d = ?", a, op, b)
	hashedAnswer := hashAnswer(fmt.Sprintf("%d", result))

	return Captcha{
		Question:  question,
		Answer:    hashedAnswer,
		ExpiresAt: time.Now().Add(1 * time.Minute),
	}
}

func hashAnswer(answer string) string {
	hash := sha256.Sum256([]byte(answer))
	return hex.EncodeToString(hash[:])
}

func VerifyCaptcha(input string, captcha Captcha) bool {

	if time.Now().After(captcha.ExpiresAt) {
		fmt.Println("Captcha expired")
		return false
	}

	return hashAnswer(input) == captcha.Answer
}
