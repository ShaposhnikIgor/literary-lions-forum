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

func GetUserIDFromSession(r *http.Request, db *sql.DB) (int, error) {

	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0, err
	}

	var userID int
	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func HandleUserPage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
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
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category)
	}

	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	pageData := models.UserPageData{
		User:       user,
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "text/html")

	// Execute the "index" template as the main entry point
	err = tmpl.ExecuteTemplate(w, "user", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}

func HandleChangeUsername(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession HandleChangeUsername: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// newUsername := r.FormValue("username")
	newUsername := strings.TrimSpace(r.FormValue("username"))

	_, err = db.Exec("UPDATE users SET username = ? WHERE id = ?", newUsername, userID)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func HandleChangePassword(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession в HandleChangePassword: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// currentPassword := r.FormValue("current_password")
	// newPassword := r.FormValue("new_password")
	// confirmPassword := r.FormValue("confirm_password")

	currentPassword := strings.TrimSpace(r.FormValue("current_password"))
	newPassword := strings.TrimSpace(r.FormValue("new_password"))
	confirmPassword := strings.TrimSpace(r.FormValue("confirm_password"))

	if newPassword != confirmPassword {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Passwords don't match")
		return
	}

	var passwordHash string
	err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		log.Printf("Error getting password_hash: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword))
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect current password")
		return
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHashedPassword, userID)
	if err != nil {
		log.Printf("Error updating password: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func ServeProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	var filePath string
	err = db.QueryRow("SELECT COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&filePath)
	if err != nil {
		log.Printf("Error getting the picture path: %v", err)
		RenderErrorPage(w, r, db, http.StatusNotFound, "Picture is not found")
		return
	}

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

	http.ServeFile(w, r, filePath)
}

func HandleUploadProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error getting ID of user from the session: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	var oldFilePath string
	err = db.QueryRow("SELECT profile_image FROM users WHERE id = ?", userID).Scan(&oldFilePath)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error getting the path to an old image: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading profile")
		return
	}

	// If no profile image is set in the database, set placeholder as default
	if oldFilePath == "" {
		oldFilePath = "assets/static/images/placeholder.png" // Default placeholder image
	}

	file, header, err := r.FormFile("profile_image")
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer file.Close()

	filePath := fmt.Sprintf("assets/static/images/uploads/%d_%s", userID, header.Filename)

	if oldFilePath != "assets/static/images/placeholder.png" {
		if err := os.Remove(oldFilePath); err != nil {
			log.Printf("Error deleting the old file: %v", err)
		}
	}

	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error saving file: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	_, err = db.Exec("UPDATE users SET profile_image = ? WHERE id = ?", filePath, userID)
	if err != nil {
		log.Printf("Error saving the path to an image in database: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func HandleChangeBio(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession HandleChangeBio: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	newBio := r.FormValue("bio")

	_, err = db.Exec("UPDATE users SET bio = ? WHERE id = ?", newBio, userID)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}
