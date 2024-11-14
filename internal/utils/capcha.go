package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

// Captcha struct holds the question, the hashed answer, and the expiration time for the captcha.
type Captcha struct {
	Question  string
	Answer    string
	ExpiresAt time.Time
}

// GenerateCaptcha creates a random captcha question with two numbers and an operation (+, -, *, /).
// It also hashes the answer and sets an expiration time for the captcha.
func GenerateCaptcha() Captcha {
	// Seed the random number generator to ensure randomness
	rand.Seed(time.Now().UnixNano())
	var a, b, result int
	var op string

	// Loop to generate a valid captcha question with a result between 0 and 9
	for {
		// Randomly generate two numbers between 0 and 9
		a = rand.Intn(10)
		b = rand.Intn(10)

		// Randomly choose an operation (addition, subtraction, multiplication, division)
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
			// For division, ensure the divisor is non-zero and the result is an integer
			if b != 0 && a%b == 0 {
				op = "/"
				result = a / b
			} else {
				// If division is not valid, continue generating a new question
				continue
			}
		}

		// Only accept questions where the result is between 0 and 9
		if result >= 0 && result <= 9 {
			break
		}
	}

	// Create the question string
	question := fmt.Sprintf("%d %s %d = ?", a, op, b)

	// Hash the answer to prevent storing it in plain text
	hashedAnswer := hashAnswer(fmt.Sprintf("%d", result))

	// Return the captcha with the question, hashed answer, and expiration time
	return Captcha{
		Question:  question,
		Answer:    hashedAnswer,
		ExpiresAt: time.Now().Add(1 * time.Minute), // Expire in 1 minute
	}
}

// hashAnswer hashes the given answer using SHA256 and returns the hashed result.
func hashAnswer(answer string) string {
	// Compute the SHA256 hash of the answer
	hash := sha256.Sum256([]byte(answer))
	// Return the hash as a hexadecimal string
	return hex.EncodeToString(hash[:])
}

// VerifyCaptcha checks whether the provided input matches the hashed answer of the captcha
// and whether the captcha has expired.
func VerifyCaptcha(input string, captcha Captcha) bool {
	// Check if the captcha has expired
	if time.Now().After(captcha.ExpiresAt) {
		fmt.Println("Captcha expired")
		return false
	}

	// Compare the hashed input with the stored answer to verify the captcha
	return hashAnswer(input) == captcha.Answer
}
