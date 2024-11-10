package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"strconv"
	"time"
)

func CreateCommentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	// Извлечение данных из формы
	postIDStr := r.FormValue("post_id")
	body := r.FormValue("body")

	// Проверка на валидность данных
	if postIDStr == "" || body == "" {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Недостаточно данных")
		return
	}

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Некорректный идентификатор поста")
		return
	}

	// Получение идентификатора пользователя из сессии
	var userID int
	cookie, err := r.Cookie("session_token")
	if err == nil {
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "Ошибка аутентификации")
			return
		}
	} else {
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "Пользователь не авторизован")
		return
	}

	// Вставка комментария в базу данных
	_, err = db.Exec("INSERT INTO comments (post_id, user_id, body, created_at) VALUES (?, ?, ?, ?)", postID, userID, body, time.Now())
	if err != nil {
		log.Printf("Ошибка при добавлении комментария: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при добавлении комментария")
		return
	}

	// Перенаправление обратно на страницу поста
	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}

func UserCommentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	// // Проверка на наличие сессии RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при добавлении комментария")пользователя
	var userID int
	// Проверка на наличие сессии пользователя
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
			if err != nil {
				log.Printf("Ошибка при получении пользователя: %v", err)
			}
		}
	} else {
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "Пользователь не авторизован")
		return
	}

	// Извлекаем комментарии пользователя с заголовками постов
	rows, err := db.Query(`
		SELECT c.id, c.post_id, c.user_id, c.body, c.created_at, p.title 
		FROM comments c 
		JOIN posts p ON c.post_id = p.id 
		WHERE c.user_id = ? 
		ORDER BY c.created_at DESC`, userID)
	if err != nil {
		log.Printf("Ошибка при извлечении комментариев: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при загрузке комментариев")
		return
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Body, &comment.CreatedAt, &comment.Title); err != nil {
			log.Printf("Ошибка при чтении комментария: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при загрузке комментариев")
			return
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при обработке результатов комментариев: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при загрузке комментариев")
		return
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

	// Передача данных в шаблон
	pageData := models.UserCommentsPageData{
		User:       user,
		Comments:   comments,
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user_comments.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки шаблона")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "user_comments", pageData)
	if err != nil {
		log.Printf("Ошибка рендеринга: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка рендеринга страницы")
		return
	}
}
