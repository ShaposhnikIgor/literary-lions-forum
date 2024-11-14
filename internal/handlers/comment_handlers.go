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

// CreateCommentHandler handles the creation of a comment on a post
func CreateCommentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the request method is POST, otherwise render an error page
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Extract form data from the request
	postIDStr := r.FormValue("post_id")  // Get the post ID from the form
	body := r.FormValue("body")          // Get the body of the comment from the form

	// Validate that both post ID and comment body are provided
	if postIDStr == "" || body == "" {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Not enough data")
		return
	}

	// Convert the post ID from string to integer
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		// If the conversion fails, render an error page
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect post identifier")
		return
	}

	// Retrieve the user ID from the session token stored in the cookie
	var userID int
	cookie, err := r.Cookie("session_token")  // Get the session token from the cookie
	if err == nil {
		// Query the database for the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			// If there’s an issue with authentication, render an error page
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "Authentication error")
			return
		}
	} else {
		// If no session cookie is found, render an unauthorized error page
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorized")
		return
	}

	// Insert the new comment into the database
	_, err = db.Exec("INSERT INTO comments (post_id, user_id, body, created_at) VALUES (?, ?, ?, ?)", postID, userID, body, time.Now())
	if err != nil {
		// If an error occurs while inserting the comment, log it and render an error page
		log.Printf("Error when adding the comment: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error when adding the comment")
		return
	}

	// Redirect the user back to the post page after successful comment creation
	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}

func UserCommentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the request method is GET, otherwise render an error page
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Check if the user session exists and retrieve the user data
	var userID int
	var user *models.User
	cookie, err := r.Cookie("session_token")  // Retrieve the session token from the cookie
	if err == nil {
		// Query the database for the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			// Retrieve user details like username, email, bio, and profile image
			err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
			if err != nil {
				log.Printf("Error when getting a user: %v", err)  // Log if there's an issue fetching user data
			}
		}
	} else {
		// If session is not found, render an unauthorized error page
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorized")
		return
	}

	// Retrieve the comments of the user along with the post titles
	rows, err := db.Query(`
		SELECT c.id, c.post_id, c.user_id, c.body, c.created_at, p.title 
		FROM comments c 
		JOIN posts p ON c.post_id = p.id 
		WHERE c.user_id = ? 
		ORDER BY c.created_at DESC`, userID)
	if err != nil {
		log.Printf("Error when getting comments: %v", err)  // Log if there’s an issue with the comment query
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
		return
	}
	defer rows.Close()

	// Collect all user comments
	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		// Scan the comment data from the query result into the Comment struct
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Body, &comment.CreatedAt, &comment.Title); err != nil {
			log.Printf("Error when reading comments: %v", err)  // Log if there's an error while reading comment data
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
			return
		}
		comments = append(comments, comment)
	}

	// Check for any error that occurred during the iteration of rows
	if err := rows.Err(); err != nil {
		log.Printf("Error rendering comments results: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
		return
	}

	// Retrieve categories for filtering or grouping the posts
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Error loading categories: %v", err)  // Log any issues with category retrieval
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close()

	// Collect all categories into a slice
	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		// Scan the category data into the Category struct
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Error reading categories: %v", err)  // Log if there’s an issue reading categories
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category)
	}

	// Check for any error during the iteration of category rows
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error rendering categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare data to be passed to the template for rendering
	pageData := models.UserCommentsPageData{
		User:       user,       // Pass the user information
		Comments:   comments,   // Pass the list of comments
		Categories: categories, // Pass the list of categories
	}

	// Parse the templates for the page
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user_comments.html")
	if err != nil {
		log.Printf("Error loading templates: %v", err)  // Log if there’s an issue loading the templates
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading templates")
		return
	}

	// Set the response content type to HTML and render the template
	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "user_comments", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)  // Log rendering issues
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Page rendering error")
		return
	}
}
