package handlers

import (
	"database/sql"
	"html/template"
	"literary-lions/internal/models"
	"literary-lions/internal/utils"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// HandleLogin handles both GET and POST requests for the login page
// It checks the login credentials, creates a session if valid, and sets a session cookie
func HandleLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the method is GET or POST; if it's neither, return a method not allowed error
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Ensure the requested URL path is "/login"; if not, render a not found error page
	if r.URL.Path != "/login" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page not found")
		return
	}

	// If the request method is GET, render the login page without any error message
	if r.Method == http.MethodGet {
		renderLoginPage(w, r, db, "")
		return
	}

	// If the request method is POST, process the form data for login
	if r.Method == http.MethodPost {
		// Get the username or email and password from the form
		username := r.FormValue("username or email")
		password := r.FormValue("password")

		// Query the database to find the user by either username or email
		var user models.User
		err := db.QueryRow("SELECT id, username, email, password_hash FROM users WHERE (username = ? OR email = ?)", username, username).
			Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash)

		// If the user is not found, render the login page with an error message
		if err != nil {
			if err == sql.ErrNoRows {
				renderLoginPage(w, r, db, "Incorrect user's name, email or password")
			} else {
				// Log the error and render an internal server error page
				log.Printf("Error searching user by name: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			}
			return
		}

		// Compare the provided password with the stored password hash
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			// If the passwords don't match, render the login page with an error message
			renderLoginPage(w, r, db, "Incorrect user's name, email or password")
			return
		}

		// Generate a new session token
		sessionToken, err := utils.CreateSessionToken()
		if err != nil {
			// If there's an error creating the session token, render an internal server error page
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session token")
			return
		}

		// Insert the session data into the database
		_, err = db.Exec("INSERT INTO sessions (user_id, session_token, created_at) VALUES (?, ?, ?)", user.ID, sessionToken, time.Now())
		if err != nil {
			// Log the error and render an internal server error page if the session insertion fails
			log.Printf("Error adding session to database: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session")
			return
		}

		// Set the session token as a secure cookie in the user's browser
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",  // Name of the cookie
			Value:    sessionToken,     // Value of the cookie (session token)
			Expires:  time.Now().Add(24 * time.Hour), // Cookie expiration time (24 hours)
			HttpOnly: true, // Prevent client-side JavaScript from accessing the cookie
		})

		// Redirect the user to the home page after successful login
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func renderLoginPage(w http.ResponseWriter, r *http.Request, db *sql.DB, errorMessage string) {
	var user *models.User

	// Fetch categories from the database to display in the login page
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// If there's an error loading categories, log the error and render an error page
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close()

	// Initialize a slice to store categories
	var categories []models.Category
	// Loop through the rows returned by the database query
	for rowsCategory.Next() {
		var category models.Category
		// Scan the category ID and name into the category struct
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Log error and render error page if there's an issue reading category data
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session token")
			return
		}
		// Append the category to the categories slice
		categories = append(categories, category)
	}

	// Check if there was any error while iterating through categories
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare the data for the login page template
	pageData := models.LoginPageData{
		Error:      errorMessage, // Any error message passed for display
		User:       user,          // User details (currently nil as it's a login page)
		Categories: categories,    // Categories fetched from the database
	}

	// Parse the HTML templates for the login page and header
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/login.html")
	if err != nil {
		// If there's an error loading the template, log it and render an error page
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the content type header for the response as HTML
	w.Header().Set("Content-Type", "text/html")

	// Execute the login template, passing in the page data
	err = tmpl.ExecuteTemplate(w, "login", pageData)
	if err != nil {
		// If there's an error rendering the page, log it and render an error page
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page")
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get the session cookie from the request
	cookie, err := r.Cookie("session_token")
	if err != nil {
		// If there's no session cookie, redirect to the homepage
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Delete the session from the database using the session token from the cookie
	_, err = db.Exec("DELETE FROM sessions WHERE session_token = ?", cookie.Value)
	if err != nil {
		// If there's an error deleting the session, render an error page
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error deleting the session")
		return
	}

	// Create a new cookie to expire the current session cookie
	cookie = &http.Cookie{
		Name:   "session_token", // Name of the cookie
		Value:  "",              // Set the value to an empty string to invalidate it
		Path:   "/",             // Path for which the cookie is valid
		MaxAge: -1,              // Set MaxAge to -1 to delete the cookie
	}
	// Set the invalidated session cookie in the user's browser
	http.SetCookie(w, cookie)

	// Redirect the user to the homepage after logging out
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
