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

func CreateCommentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Extract form data
	postIDStr := r.FormValue("post_id")
	body := strings.TrimSpace(r.FormValue("body")) // Удаляем пробелы

	// Validate data
	// if postIDStr == "" || body == "" {
	// 	RenderErrorPage(w, r, db, http.StatusBadRequest, "Not enough data")
	// 	return
	// }

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect post identifier")
		return
	}

	if postIDStr == "" || body == "" {
		http.Redirect(w, r, fmt.Sprintf("/post/%d?error=The comment text cannot be empty, please enter a comment!", postID), http.StatusSeeOther)
		return
	}

	// Get user ID from session
	var userID int
	cookie, err := r.Cookie("session_token")
	if err == nil {
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "Authentication error")
			return
		}
	} else {
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorized")
		return
	}

	// Insert comment into the database
	_, err = db.Exec("INSERT INTO comments (post_id, user_id, body, created_at) VALUES (?, ?, ?, ?)", postID, userID, body, time.Now())
	if err != nil {
		log.Printf("Error when adding the comment: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error when adding the comment")
		return
	}

	// Redirect back to the post page
	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}

func UserCommentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Check if the user session exists
	var userID int
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
			if err != nil {
				log.Printf("Error when getting a user: %v", err)
			}
		}
	} else {
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorized")
		return
	}

	// Retrieve user comments with post titles
	rows, err := db.Query(`
		SELECT c.id, c.post_id, c.user_id, c.body, c.created_at, p.title 
		FROM comments c 
		JOIN posts p ON c.post_id = p.id 
		WHERE c.user_id = ? 
		ORDER BY c.created_at DESC`, userID)
	if err != nil {
		log.Printf("Error when getting comments: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
		return
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Body, &comment.CreatedAt, &comment.Title); err != nil {
			log.Printf("Error when reading comments: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
			return
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error rendering comments results: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
		return
	}

	// Fetch categories from the database
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

	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error rendering categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Pass data to the template
	pageData := models.UserCommentsPageData{
		User:       user,
		Comments:   comments,
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user_comments.html")
	if err != nil {
		log.Printf("Error loading templates: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading templates")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "user_comments", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Page rendering error")
		return
	}
}
