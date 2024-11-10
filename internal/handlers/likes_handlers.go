package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func LikeDislikeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Метод не поддерживается like")
		return
	}

	// Extract the comment ID from the URL path (e.g., /comment_like/{id})
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid URL")
		return
	}
	targetIDStr := parts[2] // This should be the ID of the comment

	// Parse form values
	targetType := r.FormValue("target_type") // "post" or "comment"
	isLikeStr := r.FormValue("is_like")      // "true" or "false"

	// Validate input
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil || (targetType != "post" && targetType != "comment") {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Некорректные данные")
		return
	}

	isLike := isLikeStr == "true" // interpret "true" as like, otherwise dislike

	// Get user ID from session
	var userID int
	cookie, err := r.Cookie("session_token")
	if err == nil {
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "Ошибка аутентификации")
			return
		}
	} else {
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "ользователь не авторизован")
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
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при добавлении like/dislike")
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
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при обновлении like/dislike")
			return
		}
	} else {
		log.Printf("Ошибка при проверке существующего like/dislike: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при проверке like/dislike")
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Успешно обновлено"))
}

// Update this to handle both GET and POST for the comment like/dislike
func CommentLikeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Метод не поддерживается")
		return
	}

	// Extract comment ID from the URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 || pathParts[1] != "comment_like" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Страница не найдена")
		return
	}

	commentID, err := strconv.Atoi(pathParts[2])
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Неверный ID комментария")
		return
	}

	// Get user info (assuming user is already logged in through session)
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

	// Handle like/dislike
	isLike := r.FormValue("is_like") == "true"
	_, err = db.Exec(`
    INSERT OR REPLACE INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
    VALUES (?, ?, ?, ?, ?)`,
		user.ID, commentID, "comment", isLike, time.Now())
	if err != nil {
		log.Printf("Ошибка при добавлении/обновлении like/dislike для комментария: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка при обновлении like/dislike")
		return
	}

	// After processing the like/dislike, redirect back to the post page
	postID := r.FormValue("post_id")
	if postID == "" {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "ID поста не найден")
		return
	}

	// Redirect to the post page with updated comment counts
	http.Redirect(w, r, fmt.Sprintf("/post/%s", postID), http.StatusSeeOther)
}

// CountLikes returns the count of likes for a given target.
func CountLikes(db *sql.DB, targetID int, targetType string) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM likes_dislikes
		WHERE target_id = ? AND target_type = ? AND is_like = 1
	`, targetID, targetType).Scan(&count)
	return count, err
}

// CountDislikes returns the count of dislikes for a given target.
func CountDislikes(db *sql.DB, targetID int, targetType string) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM likes_dislikes
		WHERE target_id = ? AND target_type = ? AND is_like = 0
	`, targetID, targetType).Scan(&count)
	return count, err
}

func UserLikesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get the user_id from the query parameter
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid user ID")
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
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading likes")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var like models.LikeDislike
		if err := rows.Scan(&like.TargetID, &like.TargetType, &like.IsLike); err != nil {
			log.Printf("Error reading like: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading likes")
			return
		}
		likes = append(likes, like)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error processing likes result: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading likes")
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
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "user_likes", pageData)
	if err != nil {
		log.Printf("Error rendering template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page")
	}
}
