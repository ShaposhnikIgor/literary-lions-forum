package handlers

import (
	// Importing necessary packages
	"database/sql"                          // For database operations
	"encoding/base64"                       // For encoding data in base64 format
	"encoding/json"                         // For working with JSON data
	"fmt"                                   // For formatted I/O
	"html/template"                         // For rendering HTML templates
	models "literary-lions/internal/models" // Importing internal models package
	"literary-lions/internal/utils"         // Importing internal utility functions
	"log"                                   // For logging error and info messages
	"net/http"                              // For HTTP server and client functionality
	"strings"                               // For string manipulation
	"time"                                  // For working with date and time

	"golang.org/x/crypto/bcrypt" // For securely hashing passwords
)

// HandleRegistration handles user registration requests
func HandleRegistration(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var ErrorMessage string // Variable to store error messages

	// Handle GET request - serve the registration page
	if r.Method == http.MethodGet {
		serveRegistrationPage(w, r, db, ErrorMessage) // Render the registration page with any existing error message
		return
	}

	// Handle POST request - process the registration form submission
	if r.Method == http.MethodPost {
		// Extract CAPTCHA and form inputs
		captchaInput := r.FormValue("captcha")                               // Retrieve CAPTCHA input
		username := strings.TrimSpace(r.FormValue("username"))               // Trim whitespace from username
		password := strings.TrimSpace(r.FormValue("password"))               // Trim whitespace from password
		confirmPassword := strings.TrimSpace(r.FormValue("confirmPassword")) // Trim whitespace from confirmPassword
		email := strings.TrimSpace(r.FormValue("email"))                     // Trim whitespace from email

		// Validate the CAPTCHA
		captchaValid, err := validateCaptcha(r, captchaInput) // Check if the CAPTCHA is valid
		if err != nil {                                       // Handle CAPTCHA validation errors
			ErrorMessage = "Error parsing captcha"
			serveRegistrationPage(w, r, db, ErrorMessage) // Render the page with error message
			return
		}
		if !captchaValid { // If CAPTCHA is invalid, display an error
			ErrorMessage = "Incorrect respond to captcha"
			serveRegistrationPage(w, r, db, ErrorMessage)
			return
		}

		// Validate form fields
		if username == "" {
			ErrorMessage = "Username cannot be empty"
			serveRegistrationPage(w, r, db, ErrorMessage)
			return
		}
		if password == "" {
			ErrorMessage = "Password cannot be empty"
			serveRegistrationPage(w, r, db, ErrorMessage)
			return
		}
		if email == "" {
			ErrorMessage = "Email cannot be empty"
			serveRegistrationPage(w, r, db, ErrorMessage)
			return
		}
		if confirmPassword == "" {
			ErrorMessage = "ConfirmPassword cannot be empty"
			serveRegistrationPage(w, r, db, ErrorMessage)
			return
		}
		if password != confirmPassword { // Ensure passwords match
			ErrorMessage = "Passwords don't match"
			serveRegistrationPage(w, r, db, ErrorMessage)
			return
		}

		// Check if a user with the same username or email already exists
		var existingUserID int
		err = db.QueryRow("SELECT id FROM users WHERE username = ? OR email = ?", username, email).Scan(&existingUserID)
		if err != nil && err != sql.ErrNoRows { // Handle unexpected database errors
			log.Printf("Error checking existed password: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			return
		}
		if existingUserID != 0 { // If user exists, display error
			ErrorMessage = "User with this user name or email is already existed"
			serveRegistrationPage(w, r, db, ErrorMessage)
			return
		}

		// Hash the user's password securely
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil { // Handle errors in password hashing
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error of password hashing")
			return
		}

		// Insert the new user into the database
		result, err := db.Exec("INSERT INTO users (username, password_hash, email) VALUES (?, ?, ?)", username, hashedPassword, email)
		if err != nil { // Handle database insertion errors
			log.Printf("Error getting the user: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			return
		}

		// Retrieve the newly inserted user's ID
		userID, err := result.LastInsertId()
		if err != nil { // Handle errors in retrieving the user's ID
			log.Printf("Error getting user's ID: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			return
		}

		// Generate a session token for the user
		sessionToken, err := utils.CreateSessionToken()
		if err != nil { // Handle errors in token generation
			log.Printf("Error creating token session: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session")
			return
		}

		// Insert the session token into the database
		_, err = db.Exec("INSERT INTO sessions (user_id, session_token, created_at) VALUES (?, ?, ?)", userID, sessionToken, time.Now())
		if err != nil { // Handle session token storage errors
			log.Printf("Error creating token session: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session")
			return
		}

		// Set the session token as a secure cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token", // Cookie name
			Value:    sessionToken,    // Session token value
			Path:     "/",             // Path scope
			MaxAge:   3600,            // Cookie expiration in seconds
			Secure:   true,            // Secure flag ensures HTTPS usage
			HttpOnly: true,            // Prevents client-side JavaScript from accessing the cookie
		})

		// Redirect the user to the home page after successful registration
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func serveRegistrationPage(w http.ResponseWriter, r *http.Request, db *sql.DB, errorMessage string) {
	// Generate a new captcha to prevent bots from registering
	captcha := utils.GenerateCaptcha()

	// Serialize the captcha object into JSON format
	captchaJSON, err := json.Marshal(captcha)
	if err != nil {
		// Log an error and render a 500 error page if captcha generation fails
		log.Printf("Error generating captcha: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error generating captcha")
		return
	}

	// Encode the captcha JSON as a Base64 string for safe storage in cookies
	captchaBase64 := base64.StdEncoding.EncodeToString(captchaJSON)
	// Set a cookie with the encoded captcha answer for later validation
	http.SetCookie(w, &http.Cookie{
		Name:   "captcha_answer", // The name of the cookie
		Value:  captchaBase64,    // The encoded captcha data
		Path:   "/register",      // Restricts the cookie to the registration page
		MaxAge: 60,               // Cookie expiration time in seconds (1 minute)
	})

	// Declare a pointer to a User object to store information about the current user (if logged in)
	var user *models.User
	// Check if the user has a session cookie
	if sessionCookie, err := r.Cookie("session_token"); err == nil {
		var userID int
		// Retrieve the user ID associated with the session token from the database
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", sessionCookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{} // Initialize a new User object
			// Retrieve the user's details using the user ID
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				// Log an error if unable to fetch user details
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Query the database for a list of all categories
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// Log an error and render a 500 error page if category loading fails
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close() // Ensure the database rows are closed after use

	// Initialize a slice to store the categories
	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		// Scan each row into a Category object
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Log an error and render a 500 error page if reading categories fails
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		// Append the category to the list
		categories = append(categories, category)
	}

	// Prepare the data for the registration page template
	pageData := models.RegisterPageData{
		CaptchaQuestion: captcha.Question, // The captcha question to display
		User:            user,             // The current user (if logged in)
		Categories:      categories,       // The list of categories
		Error:           errorMessage,     // Any error message to display
	}

	// Parse the HTML templates for the header and the registration page
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/register.html")
	if err != nil {
		// Log an error and render a 500 error page if template loading fails
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the response content type to HTML
	w.Header().Set("Content-Type", "text/html")
	// Render the registration page template with the provided data
	if err = tmpl.ExecuteTemplate(w, "register", pageData); err != nil {
		// Log an error and render a 500 error page if template rendering fails
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
	}
}

func validateCaptcha(r *http.Request, captchaInput string) (bool, error) {
	// Retrieve the captcha answer cookie from the incoming request
	cookie, err := r.Cookie("captcha_answer")
	if err != nil {
		// Return an error if the captcha cookie is missing or expired
		return false, fmt.Errorf("captcha expired or does not exist")
	}

	// Decode the Base64-encoded captcha JSON from the cookie
	captchaJSON, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		// Log an error and return a decoding error if the process fails
		log.Printf("Error decoding captcha: %v", err)
		return false, fmt.Errorf("error decoding captcha")
	}

	// Deserialize the JSON into a Captcha object
	var captcha utils.Captcha
	if err := json.Unmarshal(captchaJSON, &captcha); err != nil {
		// Log an error and return a deserialization error if the process fails
		log.Printf("Error deserializing captcha: %v", err)
		return false, fmt.Errorf("error deserializing captcha")
	}

	// Verify the user's input against the stored captcha answer
	return utils.VerifyCaptcha(captchaInput, captcha), nil
}
