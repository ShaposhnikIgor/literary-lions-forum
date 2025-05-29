package utils

// Importing required packages:
// - "crypto/rand" provides a cryptographically secure random number generator.
// - "fmt" is used for formatting strings.
// - "io" is used for I/O operations, such as reading data streams.
import (
	"crypto/rand"
	"fmt"
	"io"
)

// CreateSessionToken generates a unique session token (UUID) and returns it as a string.
// It also returns an error if the token generation fails.
func CreateSessionToken() (string, error) {

	// Create a byte slice of length 16 to store the random data for the UUID.
	uuid := make([]byte, 16)

	// Fill the byte slice with random data using a secure random number generator.
	// `io.ReadFull` ensures that all 16 bytes are populated.
	_, err := io.ReadFull(rand.Reader, uuid)
	if err != nil {
		// If an error occurs during random data generation, return an empty string and the error.
		return "", err
	}

	// Modify specific bytes to conform to the UUID version 4 (random) standard:
	// - Set the 7th byte's four most significant bits to `0100` (binary), indicating version 4.
	uuid[6] = (uuid[6] & 0x4F) | 0x40

	// - Set the 9th byte's two most significant bits to `10` (binary), indicating the UUID variant.
	uuid[8] = (uuid[8] & 0xBF) | 0x80

	// Format the UUID bytes into a standard UUID string representation.
	// The format is 8-4-4-4-12 (e.g., xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		uuid[0:4],  // First 4 bytes (8 characters).
		uuid[4:6],  // Next 2 bytes (4 characters).
		uuid[6:8],  // Next 2 bytes (4 characters, includes version bits).
		uuid[8:10], // Next 2 bytes (4 characters, includes variant bits).
		uuid[10:],  // Remaining 6 bytes (12 characters).
	), nil
}
