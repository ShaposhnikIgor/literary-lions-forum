package handlers

import (
	"database/sql"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
)

func RenderErrorPage(w http.ResponseWriter, r *http.Request, db *sql.DB, status int, message string) {
	HandleErrorPage(w, r, db, status, message)
}

func HandleErrorPage(w http.ResponseWriter, r *http.Request, db *sql.DB, status int, message string) {
	// Set the response status
	w.WriteHeader(status)

	// Retrieve user and category data for the header
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

	// Retrieve categories
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

	// Prepare data for the template
	pageData := models.ErrorPageData{
		ErrorTitle:   http.StatusText(status),
		ErrorMessage: message,
		User:         user,
		Categories:   categories,
	}

	// Parse and render the template
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/error.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}
	if err := tmpl.ExecuteTemplate(w, "error", pageData); err != nil {
		log.Printf("Rendering error of error page: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering error of error page")
		return
	}
}
