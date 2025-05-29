package handlers

// Import necessary packages for database access, logging, HTTP handling, template rendering,
// encryption, and working with file paths.
import (
	"database/sql"                          // Provides methods to work with SQL databases.
	"fmt"                                   // Provides formatted I/O functions.
	"html/template"                         // Used for rendering HTML templates.
	"io"                                    // Provides basic I/O primitives.
	models "literary-lions/internal/models" // Imports user-defined models for the application.
	"log"                                   // Used for logging messages.
	"net/http"                              // Provides HTTP client and server implementations.
	"os"                                    // Provides functions for interacting with the operating system.
	"path/filepath"                         // Provides functions to manipulate file paths.
	"strings"                               // Contains string manipulation functions.

	"golang.org/x/crypto/bcrypt" // Provides methods for hashing and comparing passwords.
)

// GetUserIDFromSession retrieves the user ID associated with the session token from the database.
func GetUserIDFromSession(r *http.Request, db *sql.DB) (int, error) {
	// Get the session token from the cookie named "session_token".
	cookie, err := r.Cookie("session_token")
	if err != nil {
		// Return 0 and the error if the cookie is not found or invalid.
		return 0, err
	}

	var userID int
	// Query the database to find the user ID associated with the session token.
	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
	if err != nil {
		// Return 0 and the error if the session token is not found in the database.
		return 0, err
	}

	// Return the user ID and no error if the session is valid.
	return userID, nil
}

// HandleUserPage handles requests to the user page, rendering user information and available categories.
func HandleUserPage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Only allow GET requests for this handler.
	if r.Method != http.MethodGet {
		// Render an error page if the method is not GET.
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	var user *models.User // Pointer to a User object to hold user data.
	// Try to retrieve the session cookie.
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		// Query the session table to get the user ID for the session token.
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			// Initialize the User struct and fetch user details from the database.
			user = &models.User{}
			err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
			if err != nil {
				// Log an error if user details cannot be retrieved.
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Fetch all available categories from the database.
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// Log and render an error page if categories cannot be fetched.
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close() // Ensure rows are closed after processing.

	var categories []models.Category // Slice to store categories.
	// Loop through the rows and scan each category into the Category struct.
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Log and render an error page if scanning fails.
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category) // Append the category to the list.
	}

	// Check if there was an error during the iteration of rows.
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Create a struct to pass user and category data to the template.
	pageData := models.UserPageData{
		User:       user,       // The user data (can be nil if not logged in).
		Categories: categories, // The list of categories.
	}

	// Parse the templates for rendering the user page.
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user.html")
	if err != nil {
		// Log and render an error page if template parsing fails.
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the response content type to HTML.
	w.Header().Set("Content-Type", "text/html")

	// Render the user page using the parsed templates and page data.
	err = tmpl.ExecuteTemplate(w, "user", pageData)
	if err != nil {
		// Log and render an error page if rendering fails.
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}

// HandleChangeUsername handles requests to change the username of the logged-in user.
func HandleChangeUsername(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Only allow POST requests for this handler.
	if r.Method != http.MethodPost {
		// Render an error page if the method is not POST.
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Retrieve the user ID from the session.
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		// Log and render an error page if the user is not authenticated.
		log.Printf("Error when GetUserIDFromSession HandleChangeUsername: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Retrieve the new username from the form data and trim whitespace.
	newUsername := strings.TrimSpace(r.FormValue("username"))

	// Update the user's username in the database.
	_, err = db.Exec("UPDATE users SET username = ? WHERE id = ?", newUsername, userID)
	if err != nil {
		// Render an error page if the database update fails.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Redirect the user to the user page after a successful username change.
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}
func HandleChangePassword(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the HTTP method is POST; if not, return a "Method Not Allowed" error page.
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Retrieve the user ID from the session; if retrieval fails, return an "Unauthorized" error page.
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession в HandleChangePassword: %v", err) // Log the error for debugging purposes.
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Extract and sanitize the password values from the form inputs, trimming any leading/trailing whitespace.
	currentPassword := strings.TrimSpace(r.FormValue("current_password"))
	newPassword := strings.TrimSpace(r.FormValue("new_password"))
	confirmPassword := strings.TrimSpace(r.FormValue("confirm_password"))

	// Check if the new password matches the confirmation password; if not, return a "Bad Request" error page.
	if newPassword != confirmPassword {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Passwords don't match")
		return
	}

	// Declare a variable to store the user's hashed password from the database.
	var passwordHash string
	// Query the database for the user's current password hash using their user ID.
	err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		log.Printf("Error getting password_hash: %v", err) // Log the error for debugging purposes.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Compare the provided current password with the stored hashed password.
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword))
	if err != nil {
		// If the passwords do not match, return a "Bad Request" error page.
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect current password")
		return
	}

	// Generate a new hashed password from the provided new password.
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		// If password hashing fails, return an "Internal Server Error" error page.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Update the user's password hash in the database with the newly generated hash.
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHashedPassword, userID)
	if err != nil {
		log.Printf("Error updating password: %v", err) // Log the error for debugging purposes.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Redirect the user to their profile page upon successful password change.
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

// ServeProfileImage serves the profile image of a user based on their session information.
func ServeProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Retrieve the user ID from the session.
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		// If the user is not authenticated, render an error page with a 401 status.
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	var filePath string
	// Query the database to get the file path of the user's profile image.
	err = db.QueryRow("SELECT COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&filePath)
	if err != nil {
		// If the query fails, log the error and render a 404 error page.
		log.Printf("Error getting the picture path: %v", err)
		RenderErrorPage(w, r, db, http.StatusNotFound, "Picture is not found")
		return
	}

	// Extract the file extension to determine the content type.
	fileExt := filepath.Ext(filePath)
	switch fileExt {
	case ".jpg", ".jpeg":
		// Set the content type to JPEG.
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		// Set the content type to PNG.
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		// Set the content type to GIF.
		w.Header().Set("Content-Type", "image/gif")
	default:
		// If the format is unsupported, render a 415 error page.
		RenderErrorPage(w, r, db, http.StatusUnsupportedMediaType, "Format of the image is not supported")
		return
	}

	// Serve the file from the specified file path.
	http.ServeFile(w, r, filePath)
}

// HandleUploadProfileImage handles the upload and update of a user's profile image.
func HandleUploadProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the request method is POST.
	if r.Method != http.MethodPost {
		// If not, render a 405 error page.
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Retrieve the user ID from the session.
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		// If the session is invalid, log the error and render a 401 error page.
		log.Printf("Error getting ID of user from the session: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	var oldFilePath string
	// Query the database to get the current profile image file path.
	err = db.QueryRow("SELECT profile_image FROM users WHERE id = ?", userID).Scan(&oldFilePath)
	if err != nil && err != sql.ErrNoRows {
		// Log any unexpected error and render a 500 error page.
		log.Printf("Error getting the path to an old image: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading profile")
		return
	}

	// Set a default placeholder image if no profile image is currently set.
	if oldFilePath == "" {
		oldFilePath = "assets/static/images/placeholder.png"
	}

	// Retrieve the uploaded file from the request.
	file, header, err := r.FormFile("profile_image")
	if err != nil {
		// If there's an error with the upload, render a 500 error page.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer file.Close()

	// Generate a unique file path for the new profile image.
	filePath := fmt.Sprintf("assets/static/images/uploads/%d_%s", userID, header.Filename)

	// Delete the old profile image if it is not the placeholder.
	if oldFilePath != "assets/static/images/placeholder.png" {
		if err := os.Remove(oldFilePath); err != nil {
			// Log any error during the deletion process.
			log.Printf("Error deleting the old file: %v", err)
		}
	}

	// Create a new file to save the uploaded image.
	out, err := os.Create(filePath)
	if err != nil {
		// If there's an error creating the file, log it and render a 500 error page.
		log.Printf("Error saving file: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer out.Close()

	// Copy the uploaded file's content into the newly created file.
	_, err = io.Copy(out, file)
	if err != nil {
		// If there's an error during the copy, render a 500 error page.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Update the database with the new profile image file path.
	_, err = db.Exec("UPDATE users SET profile_image = ? WHERE id = ?", filePath, userID)
	if err != nil {
		// Log any error during the database update and render a 500 error page.
		log.Printf("Error saving the path to an image in database: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Redirect the user to their profile page after successful upload.
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

// HandleChangeBio updates a user's bio based on the submitted form data.
func HandleChangeBio(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the request method is POST.
	if r.Method != http.MethodPost {
		// If not, render a 405 error page.
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Retrieve the user ID from the session.
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		// Log any error and render a 401 error page.
		log.Printf("Error when GetUserIDFromSession HandleChangeBio: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Get the new bio from the submitted form data.
	newBio := r.FormValue("bio")

	// Update the user's bio in the database.
	_, err = db.Exec("UPDATE users SET bio = ? WHERE id = ?", newBio, userID)
	if err != nil {
		// If the database update fails, render a 500 error page.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Redirect the user to their profile page after successful update.
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}
