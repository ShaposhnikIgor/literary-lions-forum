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
		log.Printf("Ошибка при получении категорий: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при загрузке категорий")
		return
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt); err != nil {
			log.Printf("Ошибка при чтении категории: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при загрузке категорий")
			return
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при обработке результата: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при загрузке категорий")
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
				log.Printf("Ошибка при получении пользователя: %v", err)
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
		log.Printf("Ошибка загрузки шаблона: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки шаблона")
		return
	}

	w.Header().Set("Content-Type", "text/html")

	err = tmpl.ExecuteTemplate(w, "categories", pageData)
	if err != nil {
		log.Printf("Ошибка рендеринга: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка рендеринга страницы")
		return
	}
}
