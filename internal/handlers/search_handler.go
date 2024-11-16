package handlers

import (
	"database/sql"
	//"fmt"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func SearchHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	query := r.URL.Query().Get("query")
	category := r.URL.Query().Get("category")

	// Base query to search for posts
	var results []models.Post
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT id, title, body, created_at, category_id FROM posts WHERE (title LIKE ? OR body LIKE ?)")
	params := []interface{}{"%" + query + "%", "%" + query + "%"}

	// If a category is selected, filter the search by category
	if category != "" {
		// Convert category to integer
		categoryID, err := strconv.Atoi(category)
		if err != nil {
			// Handle invalid category format (if not a number)
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect format of category")
			return
		}

		// Add the category filter to the query
		queryBuilder.WriteString(" AND category_id = ?")
		params = append(params, categoryID)
	}

	// Execute the query
	rows, err := db.Query(queryBuilder.String(), params...)
	//fmt.Println(queryBuilder.String(), params)
	if err != nil {
		log.Printf("Error when searching: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer rows.Close()

	// Read the results
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Body, &post.CreatedAt, &post.CategoryID); err != nil {
			log.Printf("Error reading post: %v", err)
			continue
		}
		results = append(results, post)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error parsing results: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Fetch user session data (optional)
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Fetch categories from the database (to populate the category filter)
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close()

	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category)
	}

	// Check for any errors while reading category rows
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare the page data
	pageData := models.SearchResultsPageData{
		Query:      query,
		Results:    results,
		User:       user,
		Categories: categories,
	}

	// Render the search results page
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/search_results.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	err = tmpl.ExecuteTemplate(w, "search_results", pageData)
	if err != nil {
		log.Printf("Rendering page error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}
