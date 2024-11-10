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
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	if r.URL.Path != "/" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Страница не найдена")
		return
	}

	// Получение постов из базы данных
	rows, err := db.Query("SELECT id, title FROM posts ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		log.Printf("Ошибка при Получение постов из базы данных: %v", err) // Логирование детали ошибки
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка базы данных")
		return
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title); err != nil {
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при чтении данных")
			return
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при обработке запроса")
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

	// Fetch categories from the database
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Ошибка загрузки категорий: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки категорий")
		return
	}
	defer rowsCategory.Close()

	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Ошибка при чтении категории: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки категорий")
			return
		}
		categories = append(categories, category)
	}

	if err := rowsCategory.Err(); err != nil {
		log.Printf("Ошибка при обработке категорий: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки категорий")
		return
	}

	// Создаем структуру для передачи в шаблон
	pageData := models.IndexPageData{
		Posts:      posts,
		User:       user, // может быть nil, если пользователь не залогинен
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/index.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки шаблона")
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "text/html")

	// Execute the "index" template as the main entry point
	err = tmpl.ExecuteTemplate(w, "index", pageData) // specify "index" here
	if err != nil {
		log.Printf("Ошибка рендеринга: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка рендеринга страницы")
		return
	}
}
