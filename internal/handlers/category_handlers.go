package handlers

import (
	"database/sql"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
)

func CategoriesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Получение всех категорий из базы данных
	rows, err := db.Query("SELECT id, name, description, created_at FROM categories ORDER BY created_at DESC")
	if err != nil {
		log.Printf("Error getting the category: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading category")
		return
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt); err != nil {
			log.Printf("Error reading category: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading category")
			return
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error parsing the result: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading category")
		return
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
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Создаем структуру для передачи в шаблон
	pageData := struct {
		Categories []models.Category
		User       *models.User // может быть nil, если пользователь не залогинен
	}{
		Categories: categories,
		User:       user,
	}

	// Парсинг и рендеринг шаблона
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/categories.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	w.Header().Set("Content-Type", "text/html")

	err = tmpl.ExecuteTemplate(w, "categories", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}
