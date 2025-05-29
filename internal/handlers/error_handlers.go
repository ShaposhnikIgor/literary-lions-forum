package handlers

import (
	"database/sql"                          // Importing the package to interact with an SQL database
	"html/template"                         // Importing the package to parse and execute HTML templates
	models "literary-lions/internal/models" // Importing the models package from the internal project structure
	"log"                                   // Importing the logging package for error logging
	"net/http"                              // Importing the package for HTTP server and client implementations
)

// RenderErrorPage is a wrapper function to handle error page rendering.
// It forwards the request to the HandleErrorPage function for processing.
func RenderErrorPage(w http.ResponseWriter, r *http.Request, db *sql.DB, status int, message string) {
	HandleErrorPage(w, r, db, status, message)
}

// HandleErrorPage renders an error page with the appropriate status and message.
// It gathers necessary data (user session, categories) and loads templates.
func HandleErrorPage(w http.ResponseWriter, r *http.Request, db *sql.DB, status int, message string) {
	// Set the HTTP status code for the response
	w.WriteHeader(status)

	// Declare a variable to hold user data, initialized as nil
	var user *models.User

	// Attempt to retrieve the session token from cookies
	cookie, err := r.Cookie("session_token")
	if err == nil { // If the cookie exists, try to fetch associated user information
		var userID int
		// Query the database to find the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil { // If the session is valid, fetch user details
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil { // Log any errors while retrieving user information
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Fetch all categories from the database
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil { // Handle errors during category fetching
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	// Ensure the database rows are closed after processing
	defer rowsCategory.Close()

	// Create a slice to hold all retrieved categories
	var categories []models.Category
	// Iterate through the query result rows to populate categories
	for rowsCategory.Next() {
		var category models.Category
		// Scan each row into a Category struct
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil { // Handle row scan errors
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category) // Add the category to the slice
	}

	// Create a data structure to pass to the template
	pageData := models.ErrorPageData{
		ErrorTitle:   http.StatusText(status), // Human-readable text for the HTTP status code
		ErrorMessage: message,                 // Custom error message
		User:         user,                    // User data, if available
		Categories:   categories,              // Retrieved categories
	}

	// Parse the header and error page templates
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/error.html")
	if err != nil { // Handle errors during template parsing
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}
	// Execute the "error" template, injecting the prepared pageData
	if err := tmpl.ExecuteTemplate(w, "error", pageData); err != nil { // Handle errors during template execution
		log.Printf("Rendering error of error page: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering error of error page")
		return
	}
}
