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

// CreateCommentHandler handles the creation of a new comment on a post.
func CreateCommentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	// Check if the request method is POST; if not, return a "Method Not Allowed" error page.
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Extract the "post_id" and "body" fields from the form data.
	postIDStr := r.FormValue("post_id")            // Retrieve the post ID as a string from the form.
	body := strings.TrimSpace(r.FormValue("body")) // Remove any leading or trailing whitespace from the comment body.

	// Convert the post ID string to an integer.
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		// If the post ID is invalid, render a "Bad Request" error page.
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect post identifier")
		return
	}

	// Check if either the post ID or comment body is empty; if so, redirect to the post page with an error message.
	if postIDStr == "" || body == "" {
		http.Redirect(w, r, fmt.Sprintf("/post/%d?error=The comment text cannot be empty, please enter a comment!", postID), http.StatusSeeOther)
		return
	}

	// Retrieve the user ID from the session token stored in cookies.
	var userID int
	cookie, err := r.Cookie("session_token") // Retrieve the session token from the cookie.
	if err == nil {
		// Query the database to find the user ID associated with the session token.
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			// If the session token is invalid, render an "Unauthorized" error page.
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "Authentication error")
			return
		}
	} else {
		// If no session token exists, render an "Unauthorized" error page.
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorized")
		return
	}

	// Insert the new comment into the "comments" table in the database.
	_, err = db.Exec("INSERT INTO comments (post_id, user_id, body, created_at) VALUES (?, ?, ?, ?)", postID, userID, body, time.Now())
	if err != nil {
		// Log the error and render an "Internal Server Error" page if the insertion fails.
		log.Printf("Error when adding the comment: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error when adding the comment")
		return
	}

	// Redirect the user back to the post page after successfully adding the comment.
	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}

func UserCommentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure the HTTP method is GET; if not, return a 405 Method Not Allowed error
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Variable to store user ID if a session is found
	var userID int
	// Variable to hold user information
	var user *models.User
	// Attempt to retrieve the "session_token" cookie from the request
	cookie, err := r.Cookie("session_token")
	if err == nil { // If the cookie is found
		// Query the database to get the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil { // If the user ID is successfully retrieved
			// Initialize a new User object to hold the user data
			user = &models.User{}
			// Query the database to get user details based on the user ID
			err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
			if err != nil { // Log an error if user details cannot be retrieved
				log.Printf("Error when getting a user: %v", err)
			}
		}
	} else { // If no session token is found, return a 401 Unauthorized error
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorized")
		return
	}

	// Query to retrieve all comments made by the user, including the titles of the related posts
	rows, err := db.Query(`
		SELECT c.id, c.post_id, c.user_id, c.body, c.created_at, p.title 
		FROM comments c 
		JOIN posts p ON c.post_id = p.id 
		WHERE c.user_id = ? 
		ORDER BY c.created_at DESC`, userID)
	if err != nil { // Handle errors that occur during the database query
		log.Printf("Error when getting comments: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
		return
	}
	defer rows.Close() // Ensure the rows are closed after processing

	// Slice to hold the user's comments
	var comments []models.Comment
	// Loop through the result set to populate the comments slice
	for rows.Next() {
		var comment models.Comment
		// Scan the current row into the Comment structure
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Body, &comment.CreatedAt, &comment.Title); err != nil {
			// Handle any scanning errors and return a 500 Internal Server Error
			log.Printf("Error when reading comments: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
			return
		}
		// Append the successfully scanned comment to the slice
		comments = append(comments, comment)
	}

	// Check if any errors occurred during iteration over the rows
	if err := rows.Err(); err != nil {
		// Log the error and return a 500 Internal Server Error
		log.Printf("Error rendering comments results: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
		return
	}

	// Fetch categories from the database
	// Executes a SQL query to fetch the `id` and `name` of all categories from the `categories` table
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil { // Checks if there was an error executing the query
		// If an error occurred, it logs the error and renders an error page
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return // Stops the function execution
	}
	defer rowsCategory.Close() // Ensures that the rows are closed after the function exits

	// Declare a slice to hold the categories retrieved from the database
	var categories []models.Category
	// Iterates over the result set returned by the query
	for rowsCategory.Next() {
		var category models.Category
		// Scans the current row's `id` and `name` into the `category` struct
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// If there is an error reading the category data, log it and render an error page
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return // Stops further execution
		}
		// Appends the newly created category struct to the `categories` slice
		categories = append(categories, category)
	}

	// Checks if there were any errors while iterating over the rows
	if err := rowsCategory.Err(); err != nil {
		// If an error is found, log it and render an error page
		log.Printf("Error rendering categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return // Stops further execution
	}

	// Pass data to the template
	// Creates a structure to pass to the HTML template containing user data, comments, and categories
	pageData := models.UserCommentsPageData{
		User:       user,       // User data that should be displayed on the page
		Comments:   comments,   // User's comments
		Categories: categories, // List of categories to display on the page
	}

	// Parse the HTML template files for the header and the user comments page
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user_comments.html")
	if err != nil { // If there is an error loading the templates, log it and render an error page
		log.Printf("Error loading templates: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading templates")
		return // Stops the function execution
	}

	// Set the content type of the HTTP response to be HTML
	w.Header().Set("Content-Type", "text/html")
	// Executes the template with the provided data and sends it as a response to the client
	err = tmpl.ExecuteTemplate(w, "user_comments", pageData)
	if err != nil { // If an error occurs during the template rendering, log it and render an error page
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Page rendering error")
		return // Stops further execution
	}

}

// Summary of the key steps:
// 1. Database Query: Fetch categories from the database.
// 2. Error Handling: Log and render an error page if anything
// goes wrong during the database query, reading results, or processing.
// 3. Template Rendering: Load the HTML templates, populate them with user
// and category data, and render them in the response.
// This code follows standard Go practices for working with database queries,
//  handling errors, and rendering HTML templates.
