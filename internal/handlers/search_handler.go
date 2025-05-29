package handlers

import (
	"database/sql"                          // Package for SQL database interactions
	"html/template"                         // Package for rendering HTML templates
	models "literary-lions/internal/models" // Import custom data models for the application
	"log"                                   // Package for logging errors and other messages
	"net/http"                              // Package for handling HTTP requests and responses
	"strconv"                               // Package for string-to-integer conversion
	"strings"                               // Package for string manipulation
)

// SearchHandler handles the search functionality for posts based on a query and optional category filter.
func SearchHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Retrieve the search query parameter from the URL
	query := r.URL.Query().Get("query")
	// Retrieve the category filter parameter from the URL
	category := r.URL.Query().Get("category")

	// Define a slice to hold the search results
	var results []models.Post

	// Use a strings.Builder to efficiently construct the SQL query
	var queryBuilder strings.Builder
	// Base SQL query to search posts by title or body
	queryBuilder.WriteString("SELECT id, title, body, created_at, category_id FROM posts WHERE (title LIKE ? OR body LIKE ?)")
	// Add placeholders for query parameters (for search term)
	params := []interface{}{"%" + query + "%", "%" + query + "%"}

	// Check if a category filter is provided
	if category != "" {
		// Convert the category parameter to an integer
		categoryID, err := strconv.Atoi(category)
		if err != nil {
			// Handle invalid category format (e.g., if not a number)
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect format of category")
			return
		}
		// Extend the SQL query to filter by category ID
		queryBuilder.WriteString(" AND category_id = ?")
		// Append the category ID to the parameters
		params = append(params, categoryID)
	}

	// Execute the constructed SQL query with the provided parameters
	rows, err := db.Query(queryBuilder.String(), params...)
	if err != nil {
		// Log the error and render an error page if the query fails
		log.Printf("Error when searching: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	// Ensure rows are closed when the function ends
	defer rows.Close()

	// Iterate through the query results
	for rows.Next() {
		var post models.Post
		// Scan the current row into a Post struct
		if err := rows.Scan(&post.ID, &post.Title, &post.Body, &post.CreatedAt, &post.CategoryID); err != nil {
			// Log any scanning errors and continue processing remaining rows
			log.Printf("Error reading post: %v", err)
			continue
		}
		// Add the post to the results slice
		results = append(results, post)
	}

	// Check for any errors encountered during row iteration
	if err := rows.Err(); err != nil {
		log.Printf("Error parsing results: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Retrieve the user's session token from cookies (if available)
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		// Fetch the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			// Fetch user details based on the retrieved user ID
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Fetch all categories from the database to populate the category filter dropdown
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// Log the error and render an error page if category query fails
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close()

	// Define a slice to hold the retrieved categories
	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		// Scan the current row into a Category struct
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Log any scanning errors and render an error page
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		// Add the category to the categories slice
		categories = append(categories, category)
	}

	// Check for errors encountered during category row iteration
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare the data required to render the search results page
	pageData := models.SearchResultsPageData{
		Query:      query,      // Search query input by the user
		Results:    results,    // Search results to display
		User:       user,       // User information (if available)
		Categories: categories, // List of categories for filtering
	}

	// Parse the HTML templates for the header and search results
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/search_results.html")
	if err != nil {
		// Log the error and render an error page if template parsing fails
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Render the search results page with the prepared data
	err = tmpl.ExecuteTemplate(w, "search_results", pageData)
	if err != nil {
		// Log the error and render an error page if template execution fails
		log.Printf("Rendering page error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}
