package handlers

import (
	"database/sql"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"text/template"
)

// HandleIndex handles requests to the home page ("/")
// It retrieves the latest posts, user information, and categories, and renders the index template
func HandleIndex(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the request method is GET; if not, render an error page
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Ensure the requested URL path is the home page; if not, render a "not found" error page
	if r.URL.Path != "/" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page is not found")
		return
	}

	// Query the database to get the 10 most recent posts, ordered by creation date
	rows, err := db.Query("SELECT id, title FROM posts ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		// Log and render an error page if there's an issue with the query
		log.Printf("Error getting posts from database: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close() // Ensure rows are closed after use

	// Initialize a slice to store the posts
	var posts []models.Post
	// Iterate over the query results and scan the post data into the slice
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title); err != nil {
			// Render an error page if there's an issue reading data
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error reading data")
			return
		}
		posts = append(posts, post) // Append the post to the slice
	}

	// Check for any errors encountered during the iteration
	if err := rows.Err(); err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error parsing request")
		return
	}

	// Initialize a variable to hold user data
	var user *models.User
	// Check if a session token cookie exists and try to retrieve the user data from the session
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		// Query the database to get the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			// Query the database to retrieve the user's ID and username
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err) // Log error if unable to get user data
			}
		}
	}

	// Fetch categories from the database to display them in the header
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// Log and render an error page if there’s an issue loading categories
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close() // Ensure rows are closed after use

	// Initialize a slice to store categories
	var categories []models.Category
	// Iterate over the rows and scan each category into the slice
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Log and render an error page if there’s an issue reading category data
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category) // Append the category to the slice
	}

	// Check for any errors during the category iteration
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare the data to be passed to the template
	pageData := models.IndexPageData{
		Posts:      posts,      // List of posts to be displayed on the homepage
		User:       user,       // The current user (if any)
		Categories: categories, // List of categories to be displayed in the header
	}

	// Parse the HTML templates for the header and homepage
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/index.html")
	if err != nil {
		// Log and render an error page if there’s an issue loading the templates
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the content type of the response to "text/html"
	w.Header().Set("Content-Type", "text/html")

	// Execute the "index" template with the provided page data and render it to the response
	err = tmpl.ExecuteTemplate(w, "index", pageData) // Specify the "index" template here
	if err != nil {
		// Log and render an error page if there’s an issue rendering the template
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}
