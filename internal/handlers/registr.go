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

	// Handle GET request: Serve the registration page
	if r.Method == http.MethodGet {
		serveRegistrationPage(w, r, db, errorMessage)
		return
	}

	// Handle POST request: Process the registration form
	if r.Method == http.MethodPost {
		// Get CAPTCHA response and form inputs (username, password, etc.)
		captchaInput := r.FormValue("captcha")
		username := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))
		confirmPassword := strings.TrimSpace(r.FormValue("confirmPassword"))
		email := strings.TrimSpace(r.FormValue("email"))

		// Validate the CAPTCHA input to prevent bots
		captchaValid, err := validateCaptcha(r, captchaInput)
		if err != nil {
			// If CAPTCHA validation fails, display an error message
			errorMessage = "Error parsing captcha"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}
		if !captchaValid {
			// If CAPTCHA is incorrect, display an error message
			errorMessage = "Incorrect respond to captcha"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		// Validate that the necessary form fields are not empty
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

		// Check if the passwords match
		if password != confirmPassword {
			errorMessage = "Passwords don't match"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		// Check if a user with the same username or email already exists
		var existingUserID int
		err = db.QueryRow("SELECT id FROM users WHERE username = ? OR email = ?", username, email).Scan(&existingUserID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error checking existed password: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			return
		}
		if existingUserID != 0 {
			// If a user exists, display an error message
			errorMessage = "User with this user name or email is already existed"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		// Hash the password using bcrypt before storing it in the database
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error of password hashing")
			return
		}

		// Insert the new user into the database
		result, err := db.Exec("INSERT INTO users (username, password_hash, email) VALUES (?, ?, ?)", username, hashedPassword, email)
		if err != nil {
			log.Printf("Error getting the user: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			return
		}

		// Get the ID of the newly inserted user
		userID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Error getting user's ID: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			return
		}

		// Generate a session token for the newly registered user
		sessionToken, err := utils.CreateSessionToken()
		if err != nil {
			log.Printf("Error creating token session: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session")
			return
		}

		// Insert the session token into the database
		_, err = db.Exec("INSERT INTO sessions (user_id, session_token, created_at) VALUES (?, ?, ?)", userID, sessionToken, time.Now())
		if err != nil {
			log.Printf("Error creating token session: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session")
			return
		}

		// Set the session token as an HTTP cookie in the user's browser
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token", // Cookie name
			Value:    sessionToken,    // Session token as the value
			Path:     "/",             // Cookie valid for the whole site
			MaxAge:   3600,            // Cookie expires in 1 hour
			Secure:   true,            // Secure cookie (only sent over HTTPS)
			HttpOnly: true,            // HTTP-only cookie (not accessible via JavaScript)
		})

		// Redirect the user to the home page after successful registration
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// serveRegistrationPage handles rendering the registration page and preparing necessary data.
func serveRegistrationPage(w http.ResponseWriter, r *http.Request, db *sql.DB, errorMessage string) {
	// Generate a CAPTCHA for the registration form
	captcha := utils.GenerateCaptcha()

	// Marshal the CAPTCHA into JSON format for transmission
	captchaJSON, err := json.Marshal(captcha)
	if err != nil {
		log.Printf("Error generating captcha: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error generating captcha")
		return
	}

	// Encode the CAPTCHA JSON into base64 for storage in a cookie
	captchaBase64 := base64.StdEncoding.EncodeToString(captchaJSON)
	http.SetCookie(w, &http.Cookie{
		Name:   "captcha_answer", // Cookie name for storing CAPTCHA answer
		Value:  captchaBase64,    // Base64 encoded CAPTCHA JSON
		Path:   "/register",      // Cookie valid only for the registration page
		MaxAge: 60,               // Cookie expires after 60 seconds
	})

	var user *models.User
	// Check if the user has a session token cookie to retrieve the user's details
	if sessionCookie, err := r.Cookie("session_token"); err == nil {
		var userID int
		// Query the session table to get the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", sessionCookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			// Fetch user details from the users table using the user ID
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Query the categories from the database to display in the registration form
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close()

	// Store the categories in a slice
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

	// Prepare data for rendering the registration page, including CAPTCHA question, user data, and categories
	pageData := models.RegisterPageData{
		CaptchaQuestion: captcha.Question, // The CAPTCHA question to display
		User:            user,              // User data, if the user is logged in
		Categories:      categories,        // Categories to display in the form
		Error:           errorMessage,      // Error message to display, if any
	}

	// Load the registration page template
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/register.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the response content type to HTML and render the registration template
	w.Header().Set("Content-Type", "text/html")
	if err = tmpl.ExecuteTemplate(w, "register", pageData); err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
	}
}

// validateCaptcha validates the CAPTCHA input against the stored CAPTCHA answer in the cookie.
func validateCaptcha(r *http.Request, captchaInput string) (bool, error) {
	// Retrieve the CAPTCHA answer from the user's cookie
	cookie, err := r.Cookie("captcha_answer")
	if err != nil {
		return false, fmt.Errorf(" Captcha expired or is not existed")
	}

	// Decode the CAPTCHA answer from base64
	captchaJSON, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Printf("Error decoding captcha: %v", err)
		return false, fmt.Errorf("error decoding captcha")
	}

	// Unmarshal the CAPTCHA JSON into a Captcha struct
	var captcha utils.Captcha
	if err := json.Unmarshal(captchaJSON, &captcha); err != nil {
		log.Printf("Error of deserialization of captcha: %v", err)
		return false, fmt.Errorf("error of deserialization of captcha")
	}

	// Verify if the input CAPTCHA matches the stored CAPTCHA
	return utils.VerifyCaptcha(captchaInput, captcha), nil
}
