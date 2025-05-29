package handlers

// Importing necessary packages
import (
	"database/sql"                          // Provides SQL database support
	models "literary-lions/internal/models" // Imports the models package for structured data types
	"log"                                   // Used for logging errors and information
	"net/http"                              // Provides HTTP client and server implementations
	"text/template"                         // Used for parsing and rendering HTML templates
)

// HandleIndex handles requests to the root ("/") page of the web application.
func HandleIndex(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the HTTP method is GET; reject any other methods.
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Ensure the request path is exactly "/", otherwise return a 404 error.
	if r.URL.Path != "/" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page is not found")
		return
	}

	// Query the database for the 10 most recent posts, ordered by creation date.
	rows, err := db.Query("SELECT id, title FROM posts ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		// Log the error and return a 500 Internal Server Error if the query fails.
		log.Printf("Error getting posts from database: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close() // Ensure rows are closed to release database resources.

	// Create a slice to hold post data retrieved from the database.
	var posts []models.Post
	for rows.Next() { // Iterate through each row in the result set.
		var post models.Post
		// Scan the current row's data into the `post` struct.
		if err := rows.Scan(&post.ID, &post.Title); err != nil {
			// Handle any errors while reading the row.
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error reading data")
			return
		}
		// Append the post to the `posts` slice.
		posts = append(posts, post)
	}

	// Check if there was an error during iteration over rows.
	if err := rows.Err(); err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error parsing request")
		return
	}

	// Initialize a pointer for the current user, set to nil by default.
	var user *models.User
	// Attempt to retrieve the session cookie from the request.
	cookie, err := r.Cookie("session_token")
	if err == nil { // Proceed if the cookie exists.
		var userID int
		// Query the database for the user ID associated with the session token.
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil { // Proceed if the user ID is found.
			user = &models.User{} // Create a new User instance.
			// Query the database for the user's details using their ID.
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err) // Log any errors retrieving the user.
			}
		}
	}

	// Query the database for all available categories.
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// Log the error and return a 500 Internal Server Error if the query fails.
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close() // Ensure rowsCategory is closed to release database resources.

	// Create a slice to hold category data retrieved from the database.
	var categories []models.Category
	for rowsCategory.Next() { // Iterate through each row in the result set.
		var category models.Category
		// Scan the current row's data into the `category` struct.
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Handle any errors while reading the row.
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		// Append the category to the `categories` slice.
		categories = append(categories, category)
	}

	// Check if there was an error during iteration over rowsCategory.
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Create a data structure to pass information to the template.
	pageData := models.IndexPageData{
		Posts:      posts,      // List of posts to display
		User:       user,       // Currently logged-in user (if any)
		Categories: categories, // List of categories to display
	}

	// Parse the necessary HTML template files.
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/index.html")
	if err != nil {
		// Log the error and return a 500 Internal Server Error if template parsing fails.
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the Content-Type header to inform the browser of the response type.
	w.Header().Set("Content-Type", "text/html")

	// Render the "index" template with the page data.
	err = tmpl.ExecuteTemplate(w, "index", pageData) // Specify "index" as the main entry point.
	if err != nil {
		// Log the error and return a 500 Internal Server Error if template rendering fails.
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}
