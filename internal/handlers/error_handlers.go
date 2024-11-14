package handlers

import (
	"database/sql"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
)

// RenderErrorPage is a wrapper function that calls HandleErrorPage
func RenderErrorPage(w http.ResponseWriter, r *http.Request, db *sql.DB, status int, message string) {
	HandleErrorPage(w, r, db, status, message)
}

// HandleErrorPage handles the rendering of an error page with relevant user and category data
func HandleErrorPage(w http.ResponseWriter, r *http.Request, db *sql.DB, status int, message string) {
	// Set the HTTP response status code based on the provided status
	w.WriteHeader(status)

	// Retrieve the current user's data based on the session token (cookie)
	var user *models.User
	cookie, err := r.Cookie("session_token")  // Retrieve the session token from the request cookie
	if err == nil {
		var userID int
		// Query the database to get the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			// Retrieve the user details (ID and username) from the database
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)  // Log error if unable to get user data
			}
		}
	}

	// Retrieve categories from the database to display them in the header
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Error loading categories: %v", err)  // Log error if there’s an issue loading categories
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")  // Render error page
		return
	}
	defer rowsCategory.Close()  // Ensure rows are closed after processing

	var categories []models.Category
	// Iterate over the categories rows and store them in a slice
	for rowsCategory.Next() {
		var category models.Category
		// Scan the category ID and name from the database query result
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Error reading categories: %v", err)  // Log error if there’s an issue reading category data
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		// Append the category to the categories slice
		categories = append(categories, category)
	}

	// Check for any errors that occurred during the iteration over category rows
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error rendering categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare the data that will be passed to the template
	pageData := models.ErrorPageData{
		ErrorTitle:   http.StatusText(status),  // Use the status code to get the corresponding error text
		ErrorMessage: message,                  // The custom error message passed to the function
		User:         user,                      // The current user’s data (if any)
		Categories:   categories,                // The list of categories
	}

	// Parse the error page template files
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/error.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)  // Log if there’s an error loading the template files
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")  // Render error page if template loading fails
		return
	}

	// Execute the template with the prepared data and send it to the client
	if err := tmpl.ExecuteTemplate(w, "error", pageData); err != nil {
		log.Printf("Rendering error of error page: %v", err)  // Log rendering issues
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering error of error page")
		return
	}
}
