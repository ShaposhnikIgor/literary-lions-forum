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

// LikeDislikeHandler handles the like/dislike functionality for posts or comments
func LikeDislikeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the request method is POST, if not return MethodNotAllowed error
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported: like")
		return
	}

	// Extract the comment or post ID from the URL (e.g., /comment_like/{id})
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid URL")
		return
	}
	targetIDStr := parts[2] // This should be the ID of the comment or post

	// Parse form values for target type (post or comment) and like/dislike status
	targetType := r.FormValue("target_type") // "post" or "comment"
	isLikeStr := r.FormValue("is_like")      // "true" or "false"

	// Validate the parsed data, ensure targetID is a valid integer and targetType is valid
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil || (targetType != "post" && targetType != "comment") {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect data")
		return
	}

	// Convert isLikeStr to a boolean value (true for like, false for dislike)
	isLike := isLikeStr == "true"

	// Get the user ID from the session token stored in the cookie
	var userID int
	cookie, err := r.Cookie("session_token")
	if err == nil {
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "Authentication error")
			return
		}
	} else {
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Check if the user has already liked or disliked this target
	var existingID int
	err = db.QueryRow(`
        SELECT id FROM likes_dislikes
        WHERE user_id = ? AND target_id = ? AND target_type = ?
    `, userID, targetID, targetType).Scan(&existingID)

	// If no existing like/dislike, insert a new like/dislike record
	if err == sql.ErrNoRows {
		_, err = db.Exec(`
            INSERT INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
            VALUES (?, ?, ?, ?, ?)
        `, userID, targetID, targetType, isLike, time.Now())
		if err != nil {
			log.Printf("Error adding like/dislike: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error adding like/dislike")
			return
		}
	} else if err == nil {
		// If an existing like/dislike found, update it with the new like/dislike value
		_, err = db.Exec(`
            UPDATE likes_dislikes
            SET is_like = ?, created_at = ?
            WHERE id = ?
        `, isLike, time.Now(), existingID)
		if err != nil {
			log.Printf("Error when updating like/dislike: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error updating like/dislike")
			return
		}
	} else {
		// If there is an error checking for an existing like/dislike
		log.Printf("Error checking existing like/dislike: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error checking like/dislike")
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully updated"))
}

// CommentLikeHandler handles both POST and GET requests for liking/disliking comments
func CommentLikeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the request method is POST; if not, return MethodNotAllowed error
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Extract comment ID from the URL path, expecting a format like /comment_like/{id}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 || pathParts[1] != "comment_like" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page is not found")
		return
	}

	// Convert comment ID from the URL into an integer
	commentID, err := strconv.Atoi(pathParts[2])
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect ID of the comment")
		return
	}

	// Get user information from the session cookie
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

	// Determine if the request is a like or dislike action based on form data
	isLike := r.FormValue("is_like") == "true"
	// Insert or replace the like/dislike entry in the database
	_, err = db.Exec(`
    INSERT OR REPLACE INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
    VALUES (?, ?, ?, ?, ?)`,
		user.ID, commentID, "comment", isLike, time.Now())
	if err != nil {
		log.Printf("Error adding/updating like/dislike for the comment: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error updating like/dislike")
		return
	}

	// After processing the like/dislike, redirect the user back to the post page
	postID := r.FormValue("post_id")
	if postID == "" {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "ID of the post is not found")
		return
	}

	// Redirect to the post page with updated comment counts
	http.Redirect(w, r, fmt.Sprintf("/post/%s", postID), http.StatusSeeOther)
}

// CountLikes retrieves the count of likes for a given target (post or comment)
func CountLikes(db *sql.DB, targetID int, targetType string) (int, error) {
	// Query to count the number of likes (is_like = 1) for the given target
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM likes_dislikes
		WHERE target_id = ? AND target_type = ? AND is_like = 1
	`, targetID, targetType).Scan(&count)
	return count, err
}

// CountDislikes returns the count of dislikes for a given target.
func CountDislikes(db *sql.DB, targetID int, targetType string) (int, error) {
	// Variable to hold the dislike count
	var count int
	
	// Execute a query to count dislikes for the target (either post or comment)
	err := db.QueryRow(`
		SELECT COUNT(*) FROM likes_dislikes
		WHERE target_id = ? AND target_type = ? AND is_like = 0
	`, targetID, targetType).Scan(&count)

	// Return the dislike count and any error
	return count, err
}

// UserLikesHandler handles the request to display a user's liked posts and comments.
func UserLikesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get the user_id from the query parameter
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		// If the user ID is invalid, render an error page
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Variable to store user information
	var user *models.User
	
	// Attempt to retrieve the user ID from the session token cookie
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		// Query the user ID from the session
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			// Query the user's details from the database
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				// Log an error if there was a problem fetching the user details
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Retrieve liked posts and comments for the user
	var likes []models.LikeDislike
	rows, err := db.Query(`
		SELECT target_id, target_type, is_like
		FROM likes_dislikes
		WHERE user_id = ? AND is_like = true
	`, userID)
	if err != nil {
		// If there was an error fetching the likes, render an error page
		log.Printf("Error fetching user's likes: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading likes")
		return
	}
	defer rows.Close()

	// Process the result set and add liked posts/comments to the 'likes' slice
	for rows.Next() {
		var like models.LikeDislike
		if err := rows.Scan(&like.TargetID, &like.TargetType, &like.IsLike); err != nil {
			// Log an error if there was an issue reading the like data
			log.Printf("Error reading like: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading likes")
			return
		}
		likes = append(likes, like)
	}

	// Check for any errors that occurred during the iteration of the rows
	if err := rows.Err(); err != nil {
		// Log any error and render an error page
		log.Printf("Error processing likes result: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading likes")
		return
	}

	// Fetch categories from the database
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// If there was an error fetching the categories, render an error page
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close()

	// Variable to store category data
	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		// Scan each category and append it to the 'categories' slice
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Log an error if there was an issue reading category data
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category)
	}

	// Check for any errors that occurred during the iteration of category rows
	if err := rowsCategory.Err(); err != nil {
		// Log any error and render an error page
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Render the template with the user's likes and categories
	pageData := struct {
		UserID     int
		Likes      []models.LikeDislike
		User       *models.User
		Categories []models.Category
	}{
		UserID:     userID,
		Likes:      likes,
		User:       user,
		Categories: categories,
	}

	// Parse and execute the template to display the page
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user_likes.html")
	if err != nil {
		// Log error if template loading fails
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the content type as HTML and render the template
	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "user_likes", pageData)
	if err != nil {
		// Log any error that occurs during template rendering
		log.Printf("Error rendering template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page")
	}
}
