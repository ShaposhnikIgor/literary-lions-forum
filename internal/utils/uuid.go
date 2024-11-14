package utils

import (
	"crypto/rand" // Import the crypto/rand package for cryptographic random number generation
	"fmt"         // Import the fmt package for formatted I/O operations
	"io"          // Import the io package for reading from byte streams
)

// CreateSessionToken generates a random session token (UUID) and returns it as a string.
// The token is created using cryptographically secure random numbers.
func CreateSessionToken() (string, error) {

	// Create a byte slice with a length of 16 bytes to store the UUID
	uuid := make([]byte, 16)

	// Read 16 random bytes into the uuid slice using crypto/rand for secure randomness
	_, err := io.ReadFull(rand.Reader, uuid)
	if err != nil {
		// Return an empty string and the error if random byte generation fails
		return "", err
	}

	// Modify specific bits to conform to the UUID version 4 standard:
	// Set the 8th byte's two most significant bits to 10 (0x80) for version 4 UUID
	uuid[8] = (uuid[8] & 0xBF) | 0x80
	// Set the 6th byte's two most significant bits to 0100 (0x40) to indicate UUID version 4
	uuid[6] = (uuid[6] & 0x4F) | 0x40

	// Format the UUID as a string and return it
	// The format is: 8-4-4-4-12 hex characters (e.g., xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
