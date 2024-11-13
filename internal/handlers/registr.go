package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	models "literary-lions/internal/models"
	"literary-lions/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

func HandleRegistration(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var errorMessage string

	if r.Method == http.MethodGet {
		serveRegistrationPage(w, r, db, errorMessage)
		return
	}

	if r.Method == http.MethodPost {
		// Get CAPTCHA and form inputs
		captchaInput := r.FormValue("captcha")

		username := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))
		confirmPassword := strings.TrimSpace(r.FormValue("confirmPassword"))
		email := strings.TrimSpace(r.FormValue("email"))

		// Validate CAPTCHA
		captchaValid, err := validateCaptcha(r, captchaInput)
		if err != nil {
			errorMessage = "Error parsing captcha"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}
		if !captchaValid {
			errorMessage = "Incorrect respond to captcha"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		if username == "" {
			errorMessage = "Username cannot be empty"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		if password == "" {
			errorMessage = "Password cannot be empty"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		if email == "" {
			errorMessage = "Email cannot be empty"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		if confirmPassword == "" {
			errorMessage = "ConfirmPassword cannot be empty"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		// Check if passwords match
		if password != confirmPassword {
			errorMessage = "Passwords don't match"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		// Check for existing user
		var existingUserID int
		err = db.QueryRow("SELECT id FROM users WHERE username = ? OR email = ?", username, email).Scan(&existingUserID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking existed password: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			return
		}
		if existingUserID != 0 {
			errorMessage = "User with this user name or email is already existed"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error of password hashing")
			return
		}

		// Insert user into database
		result, err := db.Exec("INSERT INTO users (username, password_hash, email) VALUES (?, ?, ?)", username, hashedPassword, email)
		if err != nil {
			log.Printf("Error getting the user: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Error getting user's ID: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			return
		}

		// Create session token
		sessionToken, err := utils.CreateSessionToken()
		if err != nil {
			log.Printf("Error creating token session: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session")
			return
		}

		_, err = db.Exec("INSERT INTO sessions (user_id, session_token, created_at) VALUES (?, ?, ?)", userID, sessionToken, time.Now())
		if err != nil {
			log.Printf("Error creating token session: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session")
			return
		}

		// Set session token cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Path:     "/",
			MaxAge:   3600,
			Secure:   true,
			HttpOnly: true,
		})

		// Redirect to home page after successful registration
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func serveRegistrationPage(w http.ResponseWriter, r *http.Request, db *sql.DB, errorMessage string) {
	captcha := utils.GenerateCaptcha()

	captchaJSON, err := json.Marshal(captcha)
	if err != nil {
		log.Printf("Error generating captcha: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error generating captcha")
		return
	}

	captchaBase64 := base64.StdEncoding.EncodeToString(captchaJSON)
	http.SetCookie(w, &http.Cookie{
		Name:   "captcha_answer",
		Value:  captchaBase64,
		Path:   "/register",
		MaxAge: 60,
	})

	var user *models.User
	if sessionCookie, err := r.Cookie("session_token"); err == nil {
		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", sessionCookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

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

	pageData := models.RegisterPageData{
		CaptchaQuestion: captcha.Question,
		User:            user,
		Categories:      categories,
		Error:           errorMessage,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/register.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err = tmpl.ExecuteTemplate(w, "register", pageData); err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
	}
}

func validateCaptcha(r *http.Request, captchaInput string) (bool, error) {
	cookie, err := r.Cookie("captcha_answer")
	if err != nil {
		return false, fmt.Errorf(" Captcha expired or is not existed")
	}

	captchaJSON, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Printf("Error decoding captcha: %v", err)
		return false, fmt.Errorf("error decoding captcha")
	}

	var captcha utils.Captcha
	if err := json.Unmarshal(captchaJSON, &captcha); err != nil {
		log.Printf("Error of deserialization of captcha: %v", err)
		return false, fmt.Errorf("error of deserialization of captcha")
	}

	return utils.VerifyCaptcha(captchaInput, captcha), nil
}
