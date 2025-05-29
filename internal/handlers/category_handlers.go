package handlers

import (
	// Importing required packages for database operations, templating, logging, and HTTP handling
	"database/sql"                          // Provides SQL database interaction capabilities
	"html/template"                         // Used for rendering HTML templates
	models "literary-lions/internal/models" // Internal package containing the data models
	"log"                                   // Provides logging capabilities
	"net/http"                              // Provides HTTP client and server implementations
)

// CategoriesHandler handles the "/categories" route, displaying all categories and user session info
func CategoriesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the request method is GET; otherwise, return a "Method Not Allowed" error
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not allowed") // Render error page
		return
	}

	// Retrieve all categories from the database, ordered by creation date in descending order
	rows, err := db.Query("SELECT id, name, description, created_at FROM categories ORDER BY created_at DESC")
	if err != nil { // Handle any database query errors
		log.Printf("Error getting the category: %v", err)                                   // Log the error for debugging purposes
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading category") // Render error page
		return
	}
	defer rows.Close() // Ensure that the database rows are properly closed after use

	// Prepare a slice to store the retrieved categories
	var categories []models.Category
	// Iterate through the rows to scan the data into the category models
	for rows.Next() {
		var category models.Category // Create a variable to hold a single category's data
		// Scan the current row's data into the category struct
		if err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt); err != nil {
			log.Printf("Error reading category: %v", err)                                       // Log any scanning error
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading category") // Render error page
			return
		}
		categories = append(categories, category) // Append the category to the list
	}

	// Check if any error occurred during the iteration over rows
	if err := rows.Err(); err != nil {
		log.Printf("Error parsing the result: %v", err)                                     // Log the parsing error
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading category") // Render error page
		return
	}

	// Initialize a variable to store user data if a session exists
	var user *models.User
	// Attempt to retrieve the session token from cookies
	cookie, err := r.Cookie("session_token")
	if err == nil { // If a session token is found
		var userID int // Variable to store the user ID associated with the session
		// Query the database to find the user ID corresponding to the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil { // If the user ID is successfully retrieved
			user = &models.User{} // Initialize the user model
			// Query the database to fetch user details
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil { // Log any errors while retrieving user details
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Structure to store the categories and user data to be passed to the template
	pageData := struct {
		Categories []models.Category // List of all categories
		User       *models.User      // Logged-in user info; may be nil if no user is logged in
	}{
		Categories: categories, // Pass the retrieved categories
		User:       user,       // Pass the user data (or nil)
	}

	// Parse the necessary HTML templates for rendering the page
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/categories.html")
	if err != nil { // Handle errors during template parsing
		log.Printf("Error loading template: %v", err)                                       // Log the template error
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template") // Render error page
		return
	}

	// Set the Content-Type header to indicate an HTML response
	w.Header().Set("Content-Type", "text/html")

	// Execute the template with the prepared page data
	err = tmpl.ExecuteTemplate(w, "categories", pageData)
	if err != nil { // Handle any rendering errors
		log.Printf("Rendering error: %v", err)                                            // Log the rendering error
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error") // Render error page
		return
	}
}
