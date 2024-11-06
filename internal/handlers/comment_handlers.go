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
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Извлечение данных из формы
	postIDStr := r.FormValue("post_id")
	body := r.FormValue("body")

	// Проверка на валидность данных
	if postIDStr == "" || body == "" {
		http.Error(w, "Недостаточно данных", http.StatusBadRequest)
		return
	}

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "Некорректный идентификатор поста", http.StatusBadRequest)
		return
	}

	// Получение идентификатора пользователя из сессии
	var userID int
	cookie, err := r.Cookie("session_token")
	if err == nil {
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			http.Error(w, "Ошибка аутентификации", http.StatusUnauthorized)
			return
		}
	} else {
		http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
		return
	}

	// Вставка комментария в базу данных
	_, err = db.Exec("INSERT INTO comments (post_id, user_id, body, created_at) VALUES (?, ?, ?, ?)", postID, userID, body, time.Now())
	if err != nil {
		log.Printf("Ошибка при добавлении комментария: %v", err)
		http.Error(w, "Ошибка при добавлении комментария", http.StatusInternalServerError)
		return
	}

	// Перенаправление обратно на страницу поста
	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}

// func UserCommentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
// 	if r.Method != http.MethodGet {
// 		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	// Check user session
// 	cookie, err := r.Cookie("session_token")
// 	if err != nil {
// 		http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
// 		return
// 	}

// 	var userID int
// 	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
// 	if err != nil {
// 		http.Error(w, "Ошибка аутентификации", http.StatusUnauthorized)
// 		return
// 	}

// 	// Fetch user's comments
// 	rows, err := db.Query(`
// 		SELECT c.id, c.post_id, c.user_id, c.body, c.created_at, p.title
// 		FROM comments c
// 		JOIN posts p ON c.post_id = p.id
// 		WHERE c.user_id = ?
// 		ORDER BY c.created_at DESC`, userID)
// 	if err != nil {
// 		log.Printf("Ошибка при извлечении комментариев: %v", err)
// 		http.Error(w, "Ошибка при загрузке комментариев", http.StatusInternalServerError)
// 		return
// 	}
// 	defer rows.Close()

// 	var comments []models.Comment
// 	for rows.Next() {
// 		var comment models.Comment
// 		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Body, &comment.CreatedAt); err != nil {
// 			log.Printf("Ошибка при чтении комментария: %v", err)
// 			http.Error(w, "Ошибка при загрузке комментариев", http.StatusInternalServerError)
// 			return
// 		}
// 		comments = append(comments, comment)
// 	}

// 	// Check for any errors during row iteration
// 	if err := rows.Err(); err != nil {
// 		log.Printf("Ошибка при обработке результатов комментариев: %v", err)
// 		http.Error(w, "Ошибка при загрузке комментариев", http.StatusInternalServerError)
// 		return
// 	}

// 	// Prepare data for template
// 	pageData := struct {
// 		User     *models.User
// 		Comments []models.Comment
// 	}{
// 		User:     &models.User{ID: userID},
// 		Comments: comments,
// 	}

// 	// Render template
// 	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user_comments.html")
// 	if err != nil {
// 		log.Printf("Ошибка загрузки шаблона: %v", err)
// 		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "text/html")
// 	if err := tmpl.ExecuteTemplate(w, "user_comments", pageData); err != nil {
// 		log.Printf("Ошибка рендеринга: %v", err)
// 		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
// 		return
// 	}
// }

func UserCommentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// // Проверка на наличие сессии пользователя
	var userID int
	// cookie, err := r.Cookie("session_token")
	// if err == nil {
	// 	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
	// 	if err != nil {
	// 		http.Error(w, "Ошибка аутентификации", http.StatusUnauthorized)
	// 		return
	// 	}
	// } else {
	// 	http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
	// 	return
	// }

	// Проверка на наличие сессии пользователя
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		//var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
			if err != nil {
				log.Printf("Ошибка при получении пользователя: %v", err)
			}
		}
	} else {
		http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
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
		http.Error(w, "Ошибка при загрузке комментариев", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Body, &comment.CreatedAt, &comment.Title); err != nil {
			log.Printf("Ошибка при чтении комментария: %v", err)
			http.Error(w, "Ошибка при загрузке комментариев", http.StatusInternalServerError)
			return
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при обработке результатов комментариев: %v", err)
		http.Error(w, "Ошибка при загрузке комментариев", http.StatusInternalServerError)
		return
	}

	// Передача данных в шаблон
	pageData := models.UserCommentsPageData{
		User:     user,
		Comments: comments,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user_comments.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "user_comments", pageData)
	if err != nil {
		log.Printf("Ошибка рендеринга: %v", err)
		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
		return
	}
}
