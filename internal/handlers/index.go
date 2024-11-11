package handlers

import (
	"database/sql"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"text/template"
)

func HandleIndex(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	if r.URL.Path != "/" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page is not found")
		return
	}

	rows, err := db.Query("SELECT id, title FROM posts ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		log.Printf("Error gettingposts from database: %v", err) // Логирование детали ошибки
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title); err != nil {
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error reading data")
			return
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error parsing request")
		return
	}

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

	// Fetch categories from the database
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

	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	pageData := models.IndexPageData{
		Posts:      posts,
		User:       user,
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/index.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "text/html")

	// Execute the "index" template as the main entry point
	err = tmpl.ExecuteTemplate(w, "index", pageData) // specify "index" here
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}
