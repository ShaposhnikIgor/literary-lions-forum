package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// GetUserIDFromSession retrieves the user ID associated with the session token from the request cookies
func GetUserIDFromSession(r *http.Request, db *sql.DB) (int, error) {

	// Retrieve the session token from the request cookie
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0, err
	}

	var userID int
	// Query the database to retrieve the user ID associated with the session token
	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

// HandleUserPage handles the rendering of the user's profile page
func HandleUserPage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the method is GET, as this handler is for displaying the page only
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	var user *models.User
	// Attempt to retrieve session token from cookies
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		// Validate session token and retrieve user ID
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			// Retrieve user details based on user ID
			err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Fetch categories from the database
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close()

	var categories []models.Category
	// Iterate over categories and store them in the slice
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category)
	}

	// Check for errors encountered during rows iteration
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare data for the user profile template
	pageData := models.UserPageData{
		User:       user,
		Categories: categories,
	}

	// Parse and execute the user profile template
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "text/html")

	// Render the user page with data
	err = tmpl.ExecuteTemplate(w, "user", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}

// HandleChangeUsername handles the request to change the user's username
func HandleChangeUsername(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the method is POST, as this handler is for updating data
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Get user ID from session token
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession HandleChangeUsername: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Retrieve and sanitize the new username
	newUsername := strings.TrimSpace(r.FormValue("username"))

	// Update the user's username in the database
	_, err = db.Exec("UPDATE users SET username = ? WHERE id = ?", newUsername, userID)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Redirect to the user profile page after successful update
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func HandleChangePassword(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the method is POST, as this handler is for password change requests
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Get the user ID from the session token
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession in HandleChangePassword: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Retrieve and sanitize form values for current, new, and confirm passwords
	currentPassword := strings.TrimSpace(r.FormValue("current_password"))
	newPassword := strings.TrimSpace(r.FormValue("new_password"))
	confirmPassword := strings.TrimSpace(r.FormValue("confirm_password"))

	// Check if the new password and confirm password match
	if newPassword != confirmPassword {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Passwords don't match")
		return
	}

	var passwordHash string
	// Retrieve the existing password hash from the database
	err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		log.Printf("Error getting password_hash: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Verify the current password provided by the user
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword))
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect current password")
		return
	}

	// Hash the new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Update the user's password in the database
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHashedPassword, userID)
	if err != nil {
		log.Printf("Error updating password: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Redirect to the user profile page after successful password change
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func ServeProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get the user ID from the session token
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	var filePath string
	// Retrieve the path of the user's profile image from the database
	err = db.QueryRow("SELECT COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&filePath)
	if err != nil {
		log.Printf("Error getting the picture path: %v", err)
		RenderErrorPage(w, r, db, http.StatusNotFound, "Picture is not found")
		return
	}

	// Determine the image content type based on file extension
	fileExt := filepath.Ext(filePath)
	switch fileExt {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	default:
		RenderErrorPage(w, r, db, http.StatusUnsupportedMediaType, "Format of the image is not supported")
		return
	}

	// Serve the profile image file
	http.ServeFile(w, r, filePath)
}

func HandleUploadProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the method is POST, as this handler is for image upload requests
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Get the user ID from the session token
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error getting ID of user from the session: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	var oldFilePath string
	// Retrieve the path of the existing profile image, if any
	err = db.QueryRow("SELECT profile_image FROM users WHERE id = ?", userID).Scan(&oldFilePath)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error getting the path to an old image: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading profile")
		return
	}

	// Set a default placeholder if there is no existing profile image
	if oldFilePath == "" {
		oldFilePath = "assets/static/images/placeholder.png"
	}

	// Retrieve the uploaded file and its header
	file, header, err := r.FormFile("profile_image")
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer file.Close()

	// Define the file path for the new profile image
	filePath := fmt.Sprintf("assets/static/images/uploads/%d_%s", userID, header.Filename)

	// Remove the old profile image file if it's not the default placeholder
	if oldFilePath != "assets/static/images/placeholder.png" {
		if err := os.Remove(oldFilePath); err != nil {
			log.Printf("Error deleting the old file: %v", err)
		}
	}

	// Create a new file for the uploaded image
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error saving file: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer out.Close()

	// Copy the uploaded image data to the new file
	_, err = io.Copy(out, file)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Update the database with the new profile image path
	_, err = db.Exec("UPDATE users SET profile_image = ? WHERE id = ?", filePath, userID)
	if err != nil {
		log.Printf("Error saving the path to an image in database: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Redirect to the user profile page after successful upload
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func HandleChangeBio(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the method is POST, as this handler is for bio update requests
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Get the user ID from the session token
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession HandleChangeBio: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Retrieve the new bio from the form data
	newBio := r.FormValue("bio")

	// Update the user's bio in the database
	_, err = db.Exec("UPDATE users SET bio = ? WHERE id = ?", newBio, userID)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Redirect to the user profile page after successful bio update
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}
