package handlers

import (
	"database/sql"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"strconv"
	"time"
)

func LikeDislikeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Parse form values
	targetIDStr := r.FormValue("target_id")
	targetType := r.FormValue("target_type") // "post" or "comment"
	isLikeStr := r.FormValue("is_like")      // "true" or "false"

	// Validate input
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil || (targetType != "post" && targetType != "comment") {
		http.Error(w, "Некорректные данные", http.StatusBadRequest)
		return
	}

	isLike := isLikeStr == "true" // interpret "true" as like, otherwise dislike

	// Get user ID from session
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

	// Check if a like/dislike already exists for this target by the user
	var existingID int
	err = db.QueryRow(`
		SELECT id FROM likes_dislikes
		WHERE user_id = ? AND target_id = ? AND target_type = ?
	`, userID, targetID, targetType).Scan(&existingID)

	if err == sql.ErrNoRows {
		// If no existing entry, insert a new like/dislike
		_, err = db.Exec(`
			INSERT INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, userID, targetID, targetType, isLike, time.Now())
		if err != nil {
			log.Printf("Ошибка при добавлении like/dislike: %v", err)
			http.Error(w, "Ошибка при добавлении like/dislike", http.StatusInternalServerError)
			return
		}
	} else if err == nil {
		// If an entry exists, update it with the new like/dislike value
		_, err = db.Exec(`
			UPDATE likes_dislikes
			SET is_like = ?, created_at = ?
			WHERE id = ?
		`, isLike, time.Now(), existingID)
		if err != nil {
			log.Printf("Ошибка при обновлении like/dislike: %v", err)
			http.Error(w, "Ошибка при обновлении like/dislike", http.StatusInternalServerError)
			return
		}
	} else {
		log.Printf("Ошибка при проверке существующего like/dislike: %v", err)
		http.Error(w, "Ошибка при проверке like/dislike", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Успешно обновлено"))
}

func UserLikesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get the user_id from the query parameter
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
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

	// Retrieve liked posts and comments
	var likes []models.LikeDislike
	rows, err := db.Query(`
		SELECT target_id, target_type, is_like
		FROM likes_dislikes
		WHERE user_id = ? AND is_like = true
	`, userID)
	if err != nil {
		log.Printf("Error fetching user's likes: %v", err)
		http.Error(w, "Error loading likes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var like models.LikeDislike
		if err := rows.Scan(&like.TargetID, &like.TargetType, &like.IsLike); err != nil {
			log.Printf("Error reading like: %v", err)
			http.Error(w, "Error loading likes", http.StatusInternalServerError)
			return
		}
		likes = append(likes, like)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error processing likes result: %v", err)
		http.Error(w, "Error loading likes", http.StatusInternalServerError)
		return
	}

	// Render the template with the user's likes
	pageData := struct {
		UserID int
		Likes  []models.LikeDislike
		User   *models.User
	}{
		UserID: userID,
		Likes:  likes,
		User:   user,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user_likes.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		http.Error(w, "Error loading template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "user_likes", pageData)
	if err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// import (
// 	"literary-lions-forum/internal/models"
// 	"net/http"
// 	"strconv"
// )

// func LikePostHandler(w http.ResponseWriter, r *http.Request) {
// 	postIDStr := r.FormValue("post_id")

// 	// Convert postID from string to int
// 	postID, err := strconv.Atoi(postIDStr)
// 	if err != nil {
// 		http.Error(w, "Invalid post ID", http.StatusBadRequest)
// 		return
// 	}

// 	userID, err := models.GetCurrentUserID(r)
// 	if err != nil {
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	err = models.LikePost(postID, userID)
// 	if err != nil {
// 		http.Error(w, "Failed to like post", http.StatusInternalServerError)
// 		return
// 	}

// 	http.Redirect(w, r, "/", http.StatusSeeOther)
// }

// func DislikePostHandler(w http.ResponseWriter, r *http.Request) {
// 	// Convert postID from string to int
// 	postIDStr := r.FormValue("post_id")
// 	postID, err := strconv.Atoi(postIDStr)
// 	if err != nil {
// 		http.Error(w, "Invalid post ID", http.StatusBadRequest)
// 		return
// 	}

// 	// Get current user ID
// 	userID, err := models.GetCurrentUserID(r)
// 	if err != nil {
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 		return
// 	}

// 	// Dislike the post
// 	if err := models.DislikePost(postID, userID); err != nil {
// 		http.Error(w, "Unable to dislike post", http.StatusInternalServerError)
// 		return
// 	}

// 	// Redirect after disliking
// 	http.Redirect(w, r, "/", http.StatusSeeOther)
// }
