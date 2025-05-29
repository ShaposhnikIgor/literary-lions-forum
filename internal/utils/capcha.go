package utils

import (
	"crypto/sha256" // Provides the SHA-256 hashing algorithm.
	"encoding/hex"  // Allows encoding and decoding to and from hexadecimal format.
	"fmt"           // Provides formatted I/O functions.
	"math/rand"     // Used to generate random numbers.
	"time"          // Provides functionality for measuring and displaying time.
)

// Captcha represents a captcha structure with a question, hashed answer, and expiration time.
type Captcha struct {
	Question  string    // The human-readable captcha question (e.g., "2 + 3 = ?").
	Answer    string    // The hashed answer for the question, stored securely.
	ExpiresAt time.Time // The timestamp indicating when the captcha expires.
}

// GenerateCaptcha generates a new Captcha with a simple arithmetic question.
func GenerateCaptcha() Captcha {
	rand.Seed(time.Now().UnixNano()) // Seeds the random number generator with the current time in nanoseconds.
	var a, b, result int             // Variables to hold the operands and the result of the operation.
	var op string                    // Stores the operator for the arithmetic operation.

	// Loop until a valid captcha question with a result between 0 and 9 is generated.
	for {
		a = rand.Intn(10) // Randomly generates the first operand between 0 and 9.
		b = rand.Intn(10) // Randomly generates the second operand between 0 and 9.

		// Randomly selects an arithmetic operation.
		switch rand.Intn(4) {
		case 0: // Addition
			op = "+"
			result = a + b
		case 1: // Subtraction
			op = "-"
			result = a - b
		case 2: // Multiplication
			op = "*"
			result = a * b
		case 3: // Division (ensures no division by zero and integer result)
			if b != 0 && a%b == 0 { // Only allow division if b is not zero and a is divisible by b.
				op = "/"
				result = a / b
			} else {
				continue // Skip the iteration if division conditions are not met.
			}
		}

		// Ensures the result is a single-digit non-negative number.
		if result >= 0 && result <= 9 {
			break
		}
	}

	// Constructs the captcha question as a string (e.g., "2 + 3 = ?").
	question := fmt.Sprintf("%d %s %d = ?", a, op, b)

	// Hashes the numeric answer for secure storage and comparison.
	hashedAnswer := hashAnswer(fmt.Sprintf("%d", result))

	// Returns the Captcha struct with the question, hashed answer, and expiration time set to 1 minute from now.
	return Captcha{
		Question:  question,
		Answer:    hashedAnswer,
		ExpiresAt: time.Now().Add(1 * time.Minute),
	}
}

// hashAnswer hashes the given answer string using SHA-256 and encodes it in hexadecimal format.
func hashAnswer(answer string) string {
	hash := sha256.Sum256([]byte(answer)) // Computes the SHA-256 hash of the input string.
	return hex.EncodeToString(hash[:])    // Converts the hash to a hexadecimal string and returns it.
}

// VerifyCaptcha checks if the provided input matches the captcha answer and is not expired.
func VerifyCaptcha(input string, captcha Captcha) bool {

	// Checks if the captcha has expired by comparing the current time with the expiration time.
	if time.Now().After(captcha.ExpiresAt) {
		fmt.Println("Captcha expired") // Logs a message if the captcha has expired.
		return false                   // Returns false if the captcha is expired.
	}

	// Compares the hashed input with the stored hashed answer. Returns true if they match.
	return hashAnswer(input) == captcha.Answer
}
