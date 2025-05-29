package handlers

import (
	// Importing necessary packages for database operations, template rendering, utilities, logging, and HTTP handling.
	"database/sql"                   // Provides SQL database interaction capabilities.
	"html/template"                  // Used for rendering HTML templates.
	"literary-lions/internal/models" // Importing internal models (likely defines user and other database structures).
	"literary-lions/internal/utils"  // Importing internal utilities (likely provides helper functions like session token creation).
	"log"                            // Provides logging functionality for debugging and error reporting.
	"net/http"                       // Provides HTTP request and response handling utilities.
	"time"                           // Provides time-related utilities.

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver required for database interaction.
	"golang.org/x/crypto/bcrypt"    // Used for securely comparing password hashes.
)

// HandleLogin handles the user login process.
func HandleLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the HTTP method is allowed (only GET and POST are supported).
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		// Render an error page if the method is not supported.
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Check if the requested URL path is exactly "/login".
	if r.URL.Path != "/login" {
		// Render a 404 error page if the path is not found.
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page not found")
		return
	}

	// Handle GET requests by rendering the login page.
	if r.Method == http.MethodGet {
		// Call a helper function to render the login page, passing an empty message.
		renderLoginPage(w, r, db, "")
		return
	}

	// Handle POST requests for login submission.
	if r.Method == http.MethodPost {
		// Retrieve the username/email and password from the form data.
		username := r.FormValue("username or email") // Extract "username or email" field from the request form.
		password := r.FormValue("password")          // Extract "password" field from the request form.

		var user models.User // Declare a variable to store user information.

		// Query the database for a user with the given username or email.
		err := db.QueryRow("SELECT id, username, email, password_hash FROM users WHERE (username = ? OR email = ?)", username, username).
			Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash)

		// Handle errors during the query.
		if err != nil {
			if err == sql.ErrNoRows {
				// If no user is found, re-render the login page with an error message.
				renderLoginPage(w, r, db, "Incorrect user's name, email or password")
			} else {
				// Log the database error and render a 500 error page.
				log.Printf("Error searching user by name: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			}
			return
		}

		// Compare the provided password with the stored password hash.
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			// If the password is incorrect, re-render the login page with an error message.
			renderLoginPage(w, r, db, "Incorrect user's name, email or password")
			return
		}

		// Create a new session token for the user.
		sessionToken, err := utils.CreateSessionToken()
		if err != nil {
			// Handle errors in session token creation.
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session token")
			return
		}

		// Insert the new session into the database.
		_, err = db.Exec("INSERT INTO sessions (user_id, session_token, created_at) VALUES (?, ?, ?)", user.ID, sessionToken, time.Now())
		if err != nil {
			// Log the database error and render a 500 error page.
			log.Printf("Error adding session to database: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session")
			return
		}

		// Set a cookie with the session token to authenticate the user.
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",                // Cookie name for the session token.
			Value:    sessionToken,                   // Value of the session token.
			Expires:  time.Now().Add(24 * time.Hour), // Expiration time of the cookie (24 hours).
			HttpOnly: true,                           // Restrict cookie access to HTTP only (prevents JavaScript access).
		})

		// Redirect the user to the homepage after successful login.
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func renderLoginPage(w http.ResponseWriter, r *http.Request, db *sql.DB, errorMessage string) {
	// Define a variable to hold the current user. Here, it's initialized to nil as no user is logged in.
	var user *models.User

	// Query the database to fetch all categories, retrieving their ID and name.
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// Log an error message if the query fails and render a generic error page.
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	// Ensure the database rows are closed when the function exits, to prevent resource leaks.
	defer rowsCategory.Close()

	// Initialize a slice to store the fetched categories.
	var categories []models.Category
	// Iterate through each row in the query result.
	for rowsCategory.Next() {
		// Create a variable to hold the category data for the current row.
		var category models.Category
		// Scan the ID and name from the current row into the category variable.
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Log an error if scanning fails and render an error page.
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session token")
			return
		}
		// Append the successfully scanned category to the slice.
		categories = append(categories, category)
	}

	// Check if there was an error during row iteration (e.g., a network issue).
	if err := rowsCategory.Err(); err != nil {
		// Log the error and render a generic error page.
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare the data structure to pass to the HTML template, including error messages, user data, and categories.
	pageData := models.LoginPageData{
		Error:      errorMessage,
		User:       user,       // Always nil since user data isn't fetched in this function.
		Categories: categories, // Categories fetched from the database.
	}

	// Parse the HTML templates for rendering the login page.
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/login.html")
	if err != nil {
		// Log an error if template parsing fails and render a generic error page.
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the HTTP response content type to HTML.
	w.Header().Set("Content-Type", "text/html")
	// Render the "login" template with the page data.
	err = tmpl.ExecuteTemplate(w, "login", pageData)
	if err != nil {
		// Log an error if template rendering fails and render a generic error page.
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page")
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Attempt to retrieve the session token cookie from the request.
	cookie, err := r.Cookie("session_token")
	if err != nil {
		// Redirect the user to the homepage if the cookie is missing or invalid.
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Execute a database query to delete the session associated with the session token.
	_, err = db.Exec("DELETE FROM sessions WHERE session_token = ?", cookie.Value)
	if err != nil {
		// Log an error and render a generic error page if the deletion fails.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error deleting the session")
		return
	}

	// Create a new cookie with the same name but an empty value and a negative MaxAge to invalidate it.
	cookie = &http.Cookie{
		Name:   "session_token", // The name of the session token cookie.
		Value:  "",              // Set the value to empty, effectively clearing the token.
		Path:   "/",             // The cookie applies to the entire site.
		MaxAge: -1,              // A negative MaxAge tells the browser to delete the cookie immediately.
	}
	// Set the invalidated cookie in the HTTP response to inform the browser.
	http.SetCookie(w, cookie)

	// Redirect the user to the homepage after logout.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
