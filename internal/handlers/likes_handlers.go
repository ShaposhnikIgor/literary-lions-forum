package handlers

import (
	// Importing necessary packages
	"database/sql"                          // For interacting with the SQLite database
	"fmt"                                   // For formatted I/O operations
	"html/template"                         // For HTML templating (not directly used in this function)
	models "literary-lions/internal/models" // Importing models package (not directly used here)
	"log"                                   // For logging errors or events
	"net/http"                              // For handling HTTP requests and responses
	"strconv"                               // For converting strings to integers
	"strings"                               // For manipulating strings
	"time"                                  // For handling time and timestamps
)

// LikeDislikeHandler handles user actions for liking or disliking a post or comment
func LikeDislikeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure that the request method is POST
	if r.Method != http.MethodPost {
		// If not, render an error page with "Method Not Allowed" status
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported: like")
		return
	}

	// Split the URL path into parts (e.g., /comment_like/{id})
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		// If the URL does not contain enough parts, return a "Bad Request" error
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid URL")
		return
	}
	// Extract the comment ID (expected to be the third part of the URL)
	targetIDStr := parts[2]

	// Parse form values sent in the request
	targetType := r.FormValue("target_type") // Specifies whether the target is a "post" or "comment"
	isLikeStr := r.FormValue("is_like")      // Indicates if the action is a "like" ("true") or "dislike"

	// Validate and convert the extracted target ID from string to integer
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil || (targetType != "post" && targetType != "comment") {
		// If the ID is invalid or targetType is not "post" or "comment", return a "Bad Request" error
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect data")
		return
	}

	// Determine if the action is a "like" (true) or "dislike" (false)
	isLike := isLikeStr == "true"

	// Retrieve the user ID associated with the session token
	var userID int
	cookie, err := r.Cookie("session_token") // Get session token from the user's cookies
	if err == nil {
		// Query the database to get the user ID for the provided session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			// If the session is invalid, return an "Unauthorized" error
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "Authentication error")
			return
		}
	} else {
		// If there is no session token, return an "Unauthorized" error
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Check if a like/dislike already exists for the user and target (post/comment)
	var existingID int
	err = db.QueryRow(`
        SELECT id FROM likes_dislikes
        WHERE user_id = ? AND target_id = ? AND target_type = ?
    `, userID, targetID, targetType).Scan(&existingID)

	if err == sql.ErrNoRows {
		// If no existing record is found, insert a new like/dislike entry
		_, err = db.Exec(`
            INSERT INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
            VALUES (?, ?, ?, ?, ?)
        `, userID, targetID, targetType, isLike, time.Now())
		if err != nil {
			// Log the error and return a "Internal Server Error" response
			log.Printf("Error adding like/dislike: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error adding like/dislike")
			return
		}
	} else if err == nil {
		// If a record already exists, update it with the new like/dislike value
		_, err = db.Exec(`
            UPDATE likes_dislikes
            SET is_like = ?, created_at = ?
            WHERE id = ?
        `, isLike, time.Now(), existingID)
		if err != nil {
			// Log the error and return a "Internal Server Error" response
			log.Printf("Error when updating like/dislike: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error updating like/dislike")
			return
		}
	} else {
		// Log any error that occurred while checking for an existing record
		log.Printf("Error checking excisted like/dislike: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error checking like/dislike")
		return
	}

	// If everything succeeds, send an HTTP 200 OK response
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully updated"))
}

// Handles the like or dislike action for a comment.
// This function expects an HTTP POST request and interacts with the database to record user likes/dislikes.
// It also validates the request, handles user sessions, and redirects appropriately.
func CommentLikeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the request method is POST; otherwise, return a "Method Not Allowed" error.
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Split the URL path into parts to extract the comment ID.
	pathParts := strings.Split(r.URL.Path, "/")
	// Validate that the URL structure matches the expected format.
	if len(pathParts) < 3 || pathParts[1] != "comment_like" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page is not found")
		return
	}

	// Attempt to convert the extracted comment ID to an integer.
	commentID, err := strconv.Atoi(pathParts[2])
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect ID of the comment")
		return
	}

	// Retrieve user information based on the session token from cookies.
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil { // If the session token exists, try to retrieve the user details.
		var userID int
		// Query the sessions table to find the user ID associated with the session token.
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil { // If user ID is found, retrieve additional user information.
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err) // Log any error encountered during user retrieval.
			}
		}
	}

	// Determine whether the action is a like or a dislike based on the form value.
	isLike := r.FormValue("is_like") == "true"
	// Insert or update the like/dislike record in the `likes_dislikes` table.
	_, err = db.Exec(`
    INSERT OR REPLACE INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
    VALUES (?, ?, ?, ?, ?)`,
		user.ID, commentID, "comment", isLike, time.Now())
	if err != nil { // If there's an error during the database operation, log it and return an error page.
		log.Printf("Error adding/updating like/dislike for the comment: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error updating like/dislike")
		return
	}

	// Extract the associated post ID from the form data to redirect the user back to the post page.
	postID := r.FormValue("post_id")
	if postID == "" { // If the post ID is missing, return a "Bad Request" error.
		RenderErrorPage(w, r, db, http.StatusBadRequest, "ID of the post is not found")
		return
	}

	// Redirect the user back to the post page after processing the like/dislike action.
	http.Redirect(w, r, fmt.Sprintf("/post/%s", postID), http.StatusSeeOther)
}

// Counts the number of likes for a given target (e.g., comment or post).
// `targetID` is the ID of the target, and `targetType` specifies the type (e.g., "comment").
func CountLikes(db *sql.DB, targetID int, targetType string) (int, error) {
	var count int
	// Query the `likes_dislikes` table to count rows where `is_like` is true for the specified target.
	err := db.QueryRow(`
		SELECT COUNT(*) FROM likes_dislikes
		WHERE target_id = ? AND target_type = ? AND is_like = 1
	`, targetID, targetType).Scan(&count)
	// Return the count of likes and any error encountered during the query.
	return count, err
}

// Counts the number of dislikes for a given target (e.g., comment or post).
// `targetID` is the ID of the target, and `targetType` specifies the type (e.g., "comment").
func CountDislikes(db *sql.DB, targetID int, targetType string) (int, error) {
	var count int
	// Query the `likes_dislikes` table to count rows where `is_like` is false for the specified target.
	err := db.QueryRow(`
		SELECT COUNT(*) FROM likes_dislikes
		WHERE target_id = ? AND target_type = ? AND is_like = 0
	`, targetID, targetType).Scan(&count)
	// Return the count of dislikes and any error encountered during the query.
	return count, err
}

func UserLikesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Extract the "user_id" query parameter from the URL and convert it to an integer.
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		// If the conversion fails, render an error page with HTTP 400 (Bad Request) status.
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid user ID")
		return
	}

	// Variable to hold the authenticated user's information, if available.
	var user *models.User

	// Attempt to retrieve the "session_token" cookie from the HTTP request.
	cookie, err := r.Cookie("session_token")
	if err == nil {
		// If the cookie exists, query the database to find the associated user ID.
		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			// If a user ID is found, fetch the user's details (e.g., ID, username) from the database.
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				// Log an error if retrieving the user's information fails.
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Initialize a slice to store the user's liked posts and comments.
	var likes []models.LikeDislike

	// Query the database for likes (posts or comments) by the user.
	rows, err := db.Query(`
		SELECT target_id, target_type, is_like
		FROM likes_dislikes
		WHERE user_id = ? AND is_like = true
	`, userID)
	if err != nil {
		// Log an error and render an HTTP 500 (Internal Server Error) page if the query fails.
		log.Printf("Error fetching user's likes: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading likes")
		return
	}
	defer rows.Close() // Ensure the database rows are properly closed after processing.

	// Iterate through the query results and append each like to the `likes` slice.
	for rows.Next() {
		var like models.LikeDislike
		if err := rows.Scan(&like.TargetID, &like.TargetType, &like.IsLike); err != nil {
			// Handle errors during row scanning and render an error page.
			log.Printf("Error reading like: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading likes")
			return
		}
		likes = append(likes, like)
	}

	// Check for any errors that occurred during row iteration.
	if err := rows.Err(); err != nil {
		log.Printf("Error processing likes result: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading likes")
		return
	}

	// Query the database to fetch all available categories.
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// Log an error and render an error page if the query fails.
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close() // Ensure the database rows are properly closed after processing.

	// Initialize a slice to store the fetched categories.
	var categories []models.Category

	// Iterate through the query results and append each category to the `categories` slice.
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Handle errors during row scanning and render an error page.
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category)
	}

	// Check for any errors that occurred during row iteration.
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare data to be passed to the template for rendering the response page.
	pageData := struct {
		UserID     int                  // ID of the user whose likes are being fetched.
		Likes      []models.LikeDislike // List of likes (posts/comments) by the user.
		User       *models.User         // Authenticated user's details, if available.
		Categories []models.Category    // List of all categories for rendering on the page.
	}{
		UserID:     userID,
		Likes:      likes,
		User:       user,
		Categories: categories,
	}

	// Parse the specified template files and store the result in 'tmpl'.
	// This combines "header.html" and "user_likes.html" into a single template.
	// If there's an error during parsing, it will be stored in 'err'.
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user_likes.html")

	// Check if an error occurred while parsing the template files.
	// If 'err' is not nil, log the error message, and render an error page
	// with an HTTP 500 Internal Server Error status.
	// The RenderErrorPage function is called to inform the user of the issue and stop further execution.
	if err != nil {
		log.Printf("Error loading template: %v", err)                                       // Log the error details for debugging purposes.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template") // Respond with an error page.
		return                                                                              // Exit the function to prevent further execution with an invalid template.
	}

	// Set the "Content-Type" header of the HTTP response to "text/html".
	// This informs the browser that the response contains HTML content.
	w.Header().Set("Content-Type", "text/html")

	// Execute the "user_likes" template, passing 'pageData' to populate the template's placeholders.
	// The output is written directly to the HTTP response writer 'w'.
	// If an error occurs during template execution, it is stored in 'err'.
	err = tmpl.ExecuteTemplate(w, "user_likes", pageData)

	// Check if an error occurred while rendering the template.
	// If 'err' is not nil, log the error message, and render an error page
	// with an HTTP 500 Internal Server Error status.
	if err != nil {
		log.Printf("Error rendering template: %v", err)                                   // Log the error details for debugging purposes.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page") // Respond with an error page.
	}

}
