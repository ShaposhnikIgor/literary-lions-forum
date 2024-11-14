package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// SearchHandler handles the search functionality for posts.
func SearchHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Retrieve the search query and selected category from the URL parameters
	query := r.URL.Query().Get("query")
	category := r.URL.Query().Get("category")

	// Base query to search for posts, looking in both title and body
	var results []models.Post
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT id, title, body, created_at, category_id FROM posts WHERE (title LIKE ? OR body LIKE ?)")
	params := []interface{}{"%" + query + "%", "%" + query + "%"}

	// If a category is selected, filter the search results by category
	if category != "" {
		// Convert the category parameter to an integer
		categoryID, err := strconv.Atoi(category)
		if err != nil {
			// If category is not a valid integer, render an error page
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect format of category")
			return
		}

		// Add category filter to the query
		queryBuilder.WriteString(" AND category_id = ?")
		params = append(params, categoryID)
	}

	// Execute the search query with the constructed query string and parameters
	rows, err := db.Query(queryBuilder.String(), params...)
	fmt.Println(queryBuilder.String(), params) // Debugging the query string and parameters
	if err != nil {
		log.Printf("Error when searching: %v", err)
		// Render an error page if there's an issue executing the query
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close() // Ensure rows are closed after processing

	// Read the search results into a slice of Post structs
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Body, &post.CreatedAt, &post.CategoryID); err != nil {
			log.Printf("Error reading post: %v", err)
			continue // Skip the problematic post and continue processing others
		}
		results = append(results, post) // Add the post to the results slice
	}

	// Check for any error that occurred while processing the rows
	if err := rows.Err(); err != nil {
		log.Printf("Error parsing results: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Optionally fetch the user session data if a session token is present
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		// Query the session table to get the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			// Fetch user details from the users table
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Fetch all categories from the database to populate the category filter
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close() // Ensure rows are closed after processing

	// Store the categories in a slice
	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category) // Add the category to the list
	}

	// Check for any error that occurred while processing the category rows
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare the page data to be passed to the template
	pageData := models.SearchResultsPageData{
		Query:      query,      // The search query entered by the user
		Results:    results,    // The search results (posts)
		User:       user,       // The current logged-in user (optional)
		Categories: categories, // The list of categories for the filter
	}

	// Load and render the search results template
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/search_results.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Execute the template with the prepared page data
	err = tmpl.ExecuteTemplate(w, "search_results", pageData)
	if err != nil {
		log.Printf("Rendering page error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}
