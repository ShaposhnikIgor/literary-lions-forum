package handlers

import (
	"database/sql"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"strings"
)

func SearchHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	query := r.URL.Query().Get("query")
	category := r.URL.Query().Get("category")

	// Search logic here, e.g., search for posts or comments containing the query
	var results []models.Post
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT id, title, body, created_at FROM posts WHERE title LIKE ? OR body LIKE ?")
	params := []interface{}{"%" + query + "%", "%" + query + "%"}

	if category != "" {
		queryBuilder.WriteString(" AND category_id = ?")
		params = append(params, category)
	}

	rows, err := db.Query(queryBuilder.String(), params...)
	if err != nil {
		log.Printf("Ошибка при поиске: %v", err)
		http.Error(w, "Ошибка поиска", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Body, &post.CreatedAt); err != nil {
			log.Printf("Ошибка при чтении поста: %v", err)
			continue
		}
		results = append(results, post)
	}

	// Проверка на наличие сессии пользователя
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Ошибка при получении пользователя: %v", err)
			}
		}
	}

	// Fetch categories from the database
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Ошибка загрузки категорий: %v", err)
		http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
		return
	}
	defer rowsCategory.Close()

	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Ошибка при чтении категории: %v", err)
			http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
			return
		}
		categories = append(categories, category)
	}

	if err := rowsCategory.Err(); err != nil {
		log.Printf("Ошибка при обработке категорий: %v", err)
		http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
		return
	}

	// Create a page data object with results
	pageData := models.SearchResultsPageData{
		Query:      query,
		Results:    results,
		User:       user,
		Categories: categories,
	}

	// Render the results page
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/search_results.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	// Render the search results page
	err = tmpl.ExecuteTemplate(w, "search_results", pageData)
	if err != nil {
		log.Printf("Ошибка рендеринга страницы: %v", err)
		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
		return
	}
}
