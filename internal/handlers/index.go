package handlers

import (
	"database/sql"
	//"fmt"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"text/template"
)

func HandleIndex(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		http.Error(w, "Страница не найдена", http.StatusNotFound)
		return
	}

	// Получение постов из базы данных
	rows, err := db.Query("SELECT id, title FROM posts ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		log.Printf("Ошибка при Получение постов из базы данных: %v", err) // Логирование детали ошибки
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.Title); err != nil {
			http.Error(w, "Ошибка при чтении данных", http.StatusInternalServerError)
			return
		}
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Ошибка при обработке запроса", http.StatusInternalServerError)
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
	pageData := models.IndexPageData{
		Posts: posts,
		User:  user, // может быть nil, если пользователь не залогинен
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/index.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "text/html")

	// Execute the "index" template as the main entry point
	err = tmpl.ExecuteTemplate(w, "index", pageData) // specify "index" here
	if err != nil {
		log.Printf("Ошибка рендеринга: %v", err)
		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
		return
	}
}
