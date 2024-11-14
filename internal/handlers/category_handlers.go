package handlers

import (
	"database/sql"           // Provides a generic interface for SQL databases
	"html/template"          // For rendering HTML templates
	models "literary-lions/internal/models" // Import models package for data structures
	"log"                    // For logging errors and messages
	"net/http"               // For HTTP request and response handling
)

// CategoriesHandler handles HTTP GET requests for the categories page
func CategoriesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure only GET requests are allowed
	if r.Method != http.MethodGet {
		// If not GET, return a 405 Method Not Allowed error
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Query to retrieve all categories from the database, ordered by creation date
	rows, err := db.Query("SELECT id, name, description, created_at FROM categories ORDER BY created_at DESC")
	if err != nil {
		// Log error and display a user-friendly message on failure
		log.Printf("Error getting the category: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading category")
		return
	}
	defer rows.Close() // Ensure rows are closed after processing

	// Slice to hold the retrieved categories
	var categories []models.Category
	for rows.Next() { // Iterate over each row in the result
		var category models.Category
		// Map each row's columns to category fields
		if err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt); err != nil {
			log.Printf("Error reading category: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading category")
			return
		}
		// Add the retrieved category to the slice
		categories = append(categories, category)
	}

	// Check for errors after iterating through rows
	if err := rows.Err(); err != nil {
		log.Printf("Error parsing the result: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading category")
		return
	}

	// Check if a user session exists (i.e., user is logged in)
	var user *models.User
	cookie, err := r.Cookie("session_token") // Retrieve session token from cookies
	if err == nil {                          // If cookie exists
		var userID int
		// Retrieve the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil { // If session token is valid
			user = &models.User{} // Initialize a User struct to store user details
			// Retrieve user information (ID and username) based on user ID
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Structure to hold page data for rendering the template
	pageData := struct {
		Categories []models.Category // List of categories to display
		User       *models.User      // User data if logged in, otherwise nil
	}{
		Categories: categories,
		User:       user,
	}

	// Parse and load the required templates
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/categories.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the response content type to HTML
	w.Header().Set("Content-Type", "text/html")

	// Execute the template, injecting pageData to populate dynamic content
	err = tmpl.ExecuteTemplate(w, "categories", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}
