package handlers

import (
	"database/sql"                   // Package for interacting with SQL databases
	"fmt"                            // Package for formatted I/O operations
	"html/template"                  // Package for rendering HTML templates
	"literary-lions/internal/models" // Local package containing data models
	"log"                            // Package for logging messages
	"net/http"                       // Package for HTTP client and server implementations
	"strconv"                        // Package for converting strings to other types (e.g., integers)
	"strings"                        // Package for string manipulation
	"time"                           // Package for working with time and dates
)

// PostHandler handles requests to view and interact with a specific post.
func PostHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Only GET and POST methods are supported; others are rejected.
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		// Render an error page for unsupported HTTP methods.
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Extract the post ID from the URL path.
	pathParts := strings.Split(r.URL.Path, "/")
	// Validate the URL structure. It must include "/post/{id}".
	if len(pathParts) < 3 || pathParts[1] != "post" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page not found")
		return
	}

	// Convert the post ID from a string to an integer.
	postID, err := strconv.Atoi(pathParts[2])
	if err != nil {
		// If the ID is invalid, render an error page.
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect ID of the post")
		return
	}

	var author string       // Stores the username of the post's author
	var categoryName string // Stores the name of the post's category
	var post models.Post    // Struct to hold post details

	// SQL query to retrieve post details along with its author and category.
	query := `
		SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.id = ?`
	// Execute the query and populate the variables with the result.
	err = db.QueryRow(query, postID).Scan(
		&post.ID, &post.UserID, &author, &post.Title, &post.Body,
		&post.CategoryID, &categoryName, &post.CreatedAt,
	)
	if err != nil {
		// Handle errors for no rows or general query issues.
		if err == sql.ErrNoRows {
			RenderErrorPage(w, r, db, http.StatusNotFound, "Post not found")
		} else {
			log.Printf("Error extracting the post: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading post")
		}
		return
	}

	// Extract the "error" query parameter, if present, from the URL.
	queryURL := r.URL.Query()
	errorMessage := queryURL.Get("error")

	// Check if the user is logged in by inspecting the session cookie.
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		// Retrieve the user ID from the session table using the cookie value.
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			// Retrieve the user's details (ID and username).
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Handle POST requests for liking/disliking posts or comments.
	if r.Method == http.MethodPost {
		// Determine the target type (post or comment) and if it's a like.
		targetType := r.FormValue("target_type")
		isLike := r.FormValue("is_like") == "true"

		// Validate the comment body if applicable.
		body := strings.TrimSpace(r.FormValue("body"))
		if body == "" {
			RenderPostWithError(w, r, db, postID, "Comment cannot be empty")
			return
		}

		// Process likes/dislikes for the post.
		if targetType == "post" {
			_, err = db.Exec(`
				INSERT OR REPLACE INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
				VALUES (?, ?, ?, ?, ?)`,
				user.ID, postID, "post", isLike, time.Now())
			if err != nil {
				log.Printf("Error when adding/updating like/dislike for the post: %v", err)
			}
		}

		// Handle likes/dislikes for a comment, if a comment ID is provided.
		commentID := r.FormValue("comment_id")
		if commentID != "" {
			commentIDInt, _ := strconv.Atoi(commentID)
			if targetType == "comment" {
				_, err = db.Exec(`
					INSERT INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
					VALUES (?, ?, ?, ?, ?)
					ON DUPLICATE KEY UPDATE is_like = ?, created_at = ?`,
					user.ID, commentIDInt, "comment", isLike, time.Now(), isLike, time.Now())
				if err != nil {
					log.Printf("Error when adding/updating like/dislike for comments: %v", err)
				}
			}
		}

		// Redirect back to the same post page to refresh and show updated data.
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
		return
	}

	// Fetch comments for the post along with usernames
	// Declare a slice to hold the comments retrieved from the database.
	var comments []models.Comment

	// Define the SQL query to select comments and their associated user information.
	// The query joins the "comments" and "users" tables on "user_id",
	// filters by "post_id", and orders the results by creation time in descending order.
	commentQuery := `
SELECT c.id, c.post_id, c.user_id, u.username, c.body, c.created_at
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.post_id = ?
ORDER BY c.created_at DESC
`

	// Execute the query using the database connection and the provided post ID.
	// Retrieve the rows matching the query.
	rows, err := db.Query(commentQuery, postID)
	if err != nil {
		// Log the error and render an error page with a 500 status code if the query fails.
		log.Printf("Error extracting comments: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
		return
	}
	// Ensure the rows are closed after processing to free up resources.
	defer rows.Close()

	// Iterate over the rows to extract comment data.
	for rows.Next() {
		var comment models.Comment
		// Map the columns of the current row to the fields of the Comment model.
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Username, &comment.Body, &comment.CreatedAt); err != nil {
			// Log the error and render an error page if scanning fails.
			log.Printf("Error reading comments: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
			return
		}
		// Append the populated comment to the slice.
		comments = append(comments, comment)
	}

	// Fetch like/dislike counts for the post and comments
	// Call CountLikes to get the number of likes for the post.
	postLikes, err := CountLikes(db, postID, "post")
	if err != nil {
		// Log an error if the count query for likes fails.
		log.Printf("Error getting amount of likes for the post: %v", err)
	}

	// Call CountDislikes to get the number of dislikes for the post.
	postDislikes, err := CountDislikes(db, postID, "post")
	if err != nil {
		// Log an error if the count query for dislikes fails.
		log.Printf("Error getting amount of dislikes for the post: %v", err)
	}

	// Get like/dislike counts for each comment
	// Initialize a map to store like and dislike counts for each comment.
	commentCounts := make(map[int]models.LikeDislikeCount)

	// Loop through the retrieved comments.
	for _, comment := range comments {
		// Call CountLikes for each comment to get the number of likes.
		commentLikes, err := CountLikes(db, comment.ID, "comment")
		if err != nil {
			// Log an error if the count query for likes fails, but continue processing other comments.
			log.Printf("Error getting amount of likes for the comments %d: %v", comment.ID, err)
			continue
		}

		// Call CountDislikes for each comment to get the number of dislikes.
		commentDislikes, err := CountDislikes(db, comment.ID, "comment")
		if err != nil {
			// Log an error if the count query for dislikes fails, but continue processing other comments.
			log.Printf("Error getting amount of dislikes for the comments %d: %v", comment.ID, err)
			continue
		}

		// Store the like and dislike counts in the map using the comment ID as the key.
		commentCounts[comment.ID] = models.LikeDislikeCount{
			Likes:    commentLikes,
			Dislikes: commentDislikes,
		}
	}

	// Fetch categories from the database
	// Execute a query to retrieve all categories from the "categories" table.
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// Log the error and render an error page if the query fails.
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	// Ensure the rows are closed after processing.
	defer rowsCategory.Close()

	// Declare a slice to hold the categories retrieved from the database.
	var categories []models.Category

	// Iterate over the rows to extract category data.
	for rowsCategory.Next() {
		var category models.Category
		// Map the columns of the current row to the fields of the Category model.
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Log the error and render an error page if scanning fails.
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		// Append the populated category to the slice.
		categories = append(categories, category)
	}

	// Check for any errors encountered during row iteration.
	if err := rowsCategory.Err(); err != nil {
		// Log the error and render an error page if any issues occurred.
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Render page with updated like/dislike counts
	// Create a PostPageData struct to hold all data required for rendering the page.
	pageData := models.PostPageData{
		Post:          post,
		User:          user,
		Comments:      comments,
		PostLikes:     postLikes,
		PostDislikes:  postDislikes,
		CommentCounts: commentCounts,
		Author:        author,
		Category:      categoryName,
		Categories:    categories,
		ErrorMessage:  errorMessage,
	}

	// Parse the required HTML templates for rendering the page.
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/post.html")
	if err != nil {
		// Log the error and render an error page if template parsing fails.
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the content type of the HTTP response to HTML.
	w.Header().Set("Content-Type", "text/html")

	// Render the template using the prepared data.
	err = tmpl.ExecuteTemplate(w, "post", pageData)
	if err != nil {
		// Log the error and render an error page if template execution fails.
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page")
		return
	}
}
func AllPostsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the HTTP method is GET; if not, respond with "405 Method Not Allowed"
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Mathod is not supported")
		return
	}

	// Extract "category_id" and "user_id" query parameters from the URL
	categoryIDStr := r.URL.Query().Get("category_id")
	userIDStr := r.URL.Query().Get("user_id")

	// Initialize variables to store parsed IDs and potential errors
	var categoryID, userID int
	var errCategory, errUser error

	// Convert "category_id" to an integer if it is provided
	if categoryIDStr != "" {
		categoryID, errCategory = strconv.Atoi(categoryIDStr)
		if errCategory != nil {
			// Respond with "400 Bad Request" if the conversion fails
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid category ID")
			return
		}

		// Check if the category exists in the database
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)", categoryID).Scan(&exists)
		if err != nil || !exists {
			// Respond with "404 Not Found" if the category does not exist
			RenderErrorPage(w, r, db, http.StatusNotFound, "Category not found")
			return
		}
	}

	// Convert "user_id" to an integer if it is provided
	if userIDStr != "" {
		userID, errUser = strconv.Atoi(userIDStr)
		if errUser != nil {
			// Respond with "400 Bad Request" if the conversion fails
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid user ID")
			return
		}

		// Check if the user exists in the database
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
		if err != nil || !exists {
			// Respond with "404 Not Found" if the user does not exist
			RenderErrorPage(w, r, db, http.StatusNotFound, "User is not found")
			return
		}
	}

	// Initialize query variables for fetching posts
	var rows *sql.Rows
	var err error

	// Fetch posts based on the combination of provided parameters
	if categoryIDStr != "" && userIDStr != "" {
		// Fetch posts by both category and user, joining users and categories tables
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			WHERE p.category_id = ? AND p.user_id = ?
			ORDER BY p.created_at DESC
		`, categoryID, userID)
	} else if categoryIDStr != "" {
		// Fetch posts by category, joining users and categories tables
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			WHERE p.category_id = ?
			ORDER BY p.created_at DESC
		`, categoryID)
	} else if userIDStr != "" {
		// Fetch posts by user, joining users and categories tables
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			WHERE p.user_id = ?
			ORDER BY p.created_at DESC
		`, userID)
	} else {
		// Fetch all posts, joining users and categories tables
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			ORDER BY p.created_at DESC
		`)
	}

	// Handle any errors that occurred during the query execution
	if err != nil {
		log.Printf("Error getting posts: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading posts")
		return
	}
	defer rows.Close() // Ensure that rows are closed when the function exits

	// Limit for truncating post bodies
	const limit = 200
	var posts []models.Post // Slice to store the retrieved posts

	// Iterate through the rows and extract post data
	for rows.Next() {
		var post models.Post // Initialize a new post
		var author, categoryName string
		if err := rows.Scan(
			&post.ID, &post.UserID, &author, &post.Title, &post.Body, &post.CategoryID, &categoryName, &post.CreatedAt,
		); err != nil {
			// Handle scanning errors and respond with "500 Internal Server Error"
			log.Printf("Error extracting post's data: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error extracting post's data")
			return
		}

		// Truncate the body of the post for summary display
		post.Body = truncate(post.Body, limit)
		post.Author = author
		post.CategoryName = categoryName
		posts = append(posts, post) // Add the post to the slice
	}

	// Check for errors that occurred during row iteration
	if err := rows.Err(); err != nil {
		log.Printf("Error parsing result: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading post")
		return
	}

	// Check if a session exists by looking for the "session_token" cookie
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil { // If the cookie exists
		var sessionUserID int
		// Retrieve the user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&sessionUserID)
		if err == nil {
			user = &models.User{} // Initialize a user object
			// Fetch the user's details
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", sessionUserID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err) // Log errors if fetching user fails
			}
		}
	}

	// Fetch categories from the database
	// Query the database to retrieve all categories, fetching their ID and name.
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		// Log an error message if the query fails.
		log.Printf("Error loading categories: %v", err)
		// Render an error page indicating an internal server error.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return // Exit the function to avoid further processing.
	}
	defer rowsCategory.Close() // Ensure the query result set is closed when the function ends.

	// Define a slice to store all categories retrieved from the database.
	var categories []models.Category
	// Iterate over each row in the query result.
	for rowsCategory.Next() {
		// Create a variable to hold a single category's data.
		var category models.Category
		// Map the current row's columns to the category structure.
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			// Log an error message if reading the row fails.
			log.Printf("Error reading categories: %v", err)
			// Render an error page to inform the user.
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return // Exit the function to prevent invalid data processing.
		}
		// Append the retrieved category to the categories slice.
		categories = append(categories, category)
	}

	// Check for any errors encountered during row iteration.
	if err := rowsCategory.Err(); err != nil {
		// Log an error message if iteration errors are found.
		log.Printf("Error parsing categories: %v", err)
		// Render an error page to inform the user.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return // Exit the function to handle the error.
	}

	// Create a structure to hold data for rendering the posts page.
	pageData := models.PostsPageData{
		Posts:      posts,      // Pass the posts data (assumed to be defined elsewhere).
		User:       user,       // Include the current user data.
		Categories: categories, // Add the fetched categories.
	}

	// Parse HTML templates required to render the posts page.
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/all_posts.html")
	if err != nil {
		// Log an error message if template parsing fails.
		log.Printf("Error loading template: %v", err)
		// Render an error page to indicate the failure.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return // Exit the function to avoid further execution.
	}

	// Set the HTTP header to indicate the response content type is HTML.
	w.Header().Set("Content-Type", "text/html")

	// Execute the parsed template using the page data.
	err = tmpl.ExecuteTemplate(w, "all_posts", pageData)
	if err != nil {
		// Log an error message if rendering the template fails.
		log.Printf("Rendering error: %v", err)
		// Render an error page to inform the user.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return // Exit the function to handle the error.
	}
}

// Function to truncate a string if its length exceeds the given limit.
func truncate(text string, limit int) string {
	// Check if the string length exceeds the specified limit.
	if len(text) > limit {
		// Return a substring up to the limit, appending "..." to indicate truncation.
		return text[:limit] + "..."
	}
	return text // Return the original string if it is within the limit.
}

// NewPostHandler handles the creation of new posts. It supports both GET and POST methods.
// GET displays the post creation page, while POST processes the creation of a new post.
func NewPostHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the HTTP method is GET (used for rendering the new post page).
	if r.Method == http.MethodGet {

		// Variable to hold the user information, initialized to nil.
		var user *models.User

		// Attempt to retrieve the "session_token" cookie from the request.
		cookie, err := r.Cookie("session_token")
		if err == nil { // If the cookie exists
			var userID int
			// Query the database to get the user ID associated with the session token.
			err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
			if err == nil { // If the session is valid
				user = &models.User{}
				// Retrieve user details from the database using the user ID.
				err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
				if err != nil {
					// Log the error if user details cannot be fetched.
					log.Printf("Error getting the user: %v", err)
				}
			}
		} else {
			// If no valid session, render an unauthorized error page.
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
			return
		}

		// Fetch all categories from the database.
		rows, err := db.Query("SELECT id, name FROM categories")
		if err != nil {
			// Log the error and render a server error page if categories cannot be fetched.
			log.Printf("Error loading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		defer rows.Close() // Ensure the rows are closed after processing.

		var categories []models.Category
		// Iterate through the rows to build a list of categories.
		for rows.Next() {
			var category models.Category
			if err := rows.Scan(&category.ID, &category.Name); err != nil {
				// Log and render an error page if there is an issue reading categories.
				log.Printf("Error reading categories: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
				return
			}
			categories = append(categories, category)
		}

		// Check if there were any errors during row iteration.
		if err := rows.Err(); err != nil {
			log.Printf("Error parsing categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}

		// Prepare the data for the new post page, including user and categories.
		pageData := models.NewPostPageData{
			User:       user,
			Categories: categories,
		}

		// Load the HTML templates for rendering the new post page.
		tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/new_post.html")
		if err != nil {
			// Log and render an error page if the templates cannot be loaded.
			log.Printf("Error loading template: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
			return
		}

		// Set the Content-Type header for the response to HTML.
		w.Header().Set("Content-Type", "text/html")

		// Execute the template with the prepared page data.
		err = tmpl.ExecuteTemplate(w, "new_post", pageData)
		if err != nil {
			// Log and render an error page if there is an issue rendering the template.
			log.Printf("Rendering error: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
			return
		}
		return // Exit after handling the GET request.
	}

	// Check if the HTTP method is POST (used for submitting a new post).
	if r.Method == http.MethodPost {
		// Attempt to retrieve the "session_token" cookie to authenticate the user.
		cookie, err := r.Cookie("session_token")
		if err != nil {
			// If no valid session, render an unauthorized error page.
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
			return
		}

		var userID int
		// Query the database to get the user ID associated with the session token.
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			// If the session is invalid, render an unauthorized error page.
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
			return
		}

		// Parse the form data submitted with the POST request.
		err = r.ParseForm()
		if err != nil {
			// If the form cannot be parsed, render a bad request error page.
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Error parsing the form")
			return
		}

		// Retrieve and trim the form values for title, body, and category ID.
		title := strings.TrimSpace(r.FormValue("title"))
		body := strings.TrimSpace(r.FormValue("body"))
		categoryIDStr := r.FormValue("category_id")

		// Validation
		if title == "" || body == "" {
			// Check if the title or body of the post is empty.
			// If either is empty, reload the page with an error message.

			var user *models.User
			cookie, err := r.Cookie("session_token")
			// Attempt to retrieve the "session_token" cookie to identify the user.

			if err == nil {
				// If the cookie exists, try to fetch the associated user information.

				var userID int
				err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
				// Query the `sessions` table to retrieve the user ID linked to the session token.

				if err == nil {
					// If the user ID is retrieved successfully, fetch additional user details.

					user = &models.User{}
					err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).
						Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
					// Query the `users` table to load the user's profile details, using `COALESCE` for optional fields.

					if err != nil {
						// Log an error if user details could not be retrieved.
						log.Printf("Error getting the user: %v", err)
					}
				}
			} else {
				// If the cookie is missing or invalid, render an unauthorized error page.
				RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
				return
			}

			// Fetch categories from the database.
			rowsCategory, err := db.Query("SELECT id, name FROM categories")
			// Query the `categories` table to load all available post categories.

			if err != nil {
				// Log an error if the query fails and render an internal server error page.
				log.Printf("Error loading categories: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
				return
			}
			defer rowsCategory.Close()
			// Ensure the database rows are closed properly after processing.

			var categories []models.Category
			for rowsCategory.Next() {
				// Iterate over the results of the `categories` query.

				var category models.Category
				if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
					// Read each category's ID and name into a `Category` struct.
					// Log an error and render a server error page if scanning fails.
					log.Printf("Error reading categories: %v", err)
					RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
					return
				}
				categories = append(categories, category)
				// Append the successfully scanned category to the list of categories.
			}

			if err := rowsCategory.Err(); err != nil {
				// Check for errors encountered during the iteration over query results.
				// If an error occurred, log it and render a server error page.
				log.Printf("Error parsing categories: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
				return
			}

			pageData := models.NewPostPageData{
				User:         user,
				ErrorMessage: "All fields are required and cannot be empty.",
				Categories:   categories,
			}
			// Prepare the data to render the new post page, including the user, an error message, and the categories.

			tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/new_post.html")
			// Load the templates required for rendering the "New Post" page.

			if err != nil {
				// Log an error if the templates cannot be loaded and render an internal server error page.
				log.Printf("Error loading template: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
				return
			}

			w.Header().Set("Content-Type", "text/html")
			// Set the response content type to HTML.

			tmpl.ExecuteTemplate(w, "new_post", pageData)
			// Render the "New Post" template with the prepared data.
			return
		}

		// Convert the category ID from a string to an integer.
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			// If the conversion fails, render a 404 error page indicating an invalid category ID.
			RenderErrorPage(w, r, db, http.StatusNotFound, "Incorrect ID of category")
			return
		}

		result, err := db.Exec("INSERT INTO posts (user_id, title, body, category_id, created_at) VALUES (?, ?, ?, ?, ?)",
			userID, title, body, categoryID, time.Now())
		// Insert the new post into the `posts` table, associating it with the user and category.

		if err != nil {
			// Log an error if the insertion fails and render an internal server error page.
			log.Printf("Error creating the post: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating the post")
			return
		}

		postID, err := result.LastInsertId()
		// Retrieve the ID of the newly created post.

		if err != nil {
			// If the post ID cannot be fetched, render an internal server error page.
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error getting ID of post")
			return
		}

		// Redirect the user to the page displaying the newly created post.
		http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
	}
}

func RenderPostWithError(w http.ResponseWriter, r *http.Request, db *sql.DB, postID int, errorMessage string) {
	// Declare variables to hold data for the post and related information.
	var author string       // To store the username of the post's author.
	var categoryName string // To store the name of the post's category.
	var post models.Post    // Struct to hold the main post data.

	// SQL query to retrieve the post details, including author and category information.
	query := `
        SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
        FROM posts p
        JOIN users u ON p.user_id = u.id
        JOIN categories c ON p.category_id = c.id
        WHERE p.id = ?`

	// Execute the query and scan the results into the respective variables.
	err := db.QueryRow(query, postID).Scan(
		&post.ID, &post.UserID, &author, &post.Title, &post.Body,
		&post.CategoryID, &categoryName, &post.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// If no rows are found, render a 404 error page.
			RenderErrorPage(w, r, db, http.StatusNotFound, "Post not found")
		} else {
			// For any other error, log it and render a 500 error page.
			log.Printf("Error extracting the post: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading post")
		}
		return
	}

	// Check if a user is logged in by validating the session cookie.
	var user *models.User // Pointer to a user struct to hold the logged-in user details.
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int // Variable to store the user ID from the session.
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{} // Initialize the user struct.
			// Fetch user details based on the user ID from the session.
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Handle form submissions for likes/dislikes.
	if r.Method == http.MethodPost {
		targetType := r.FormValue("target_type")   // Target type (post or comment).
		isLike := r.FormValue("is_like") == "true" // Determine if it's a like.

		body := strings.TrimSpace(r.FormValue("body")) // Get and trim the comment body.
		if body == "" {
			// If the comment body is empty, re-render the post with an error message.
			RenderPostWithError(w, r, db, postID, "Comment cannot be empty")
			return
		}

		// Handle like/dislike for a post.
		if targetType == "post" {
			_, err = db.Exec(`
            INSERT OR REPLACE INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
            VALUES (?, ?, ?, ?, ?)`,
				user.ID, postID, "post", isLike, time.Now())
			if err != nil {
				log.Printf("Error when adding/updating like/dislike for the post: %v", err)
			}
		}

		// Handle like/dislike for a comment, if provided.
		commentID := r.FormValue("comment_id")
		if commentID != "" {
			commentIDInt, _ := strconv.Atoi(commentID) // Convert comment ID to an integer.
			if targetType == "comment" {
				_, err = db.Exec(`
                    INSERT INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
                    VALUES (?, ?, ?, ?, ?)
                    ON DUPLICATE KEY UPDATE is_like = ?, created_at = ?`,
					user.ID, commentIDInt, "comment", isLike, time.Now(), isLike, time.Now())
				if err != nil {
					log.Printf("Error when adding/updating like/dislike for comments: %v", err)
				}
			}
		}

		// Redirect back to the same post page to refresh and display updated information.
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
		return
	}

	// Retrieve comments for the post, including the author's username for each comment.
	var comments []models.Comment // Slice to store comments for the post.
	commentQuery := `
    SELECT c.id, c.post_id, c.user_id, u.username, c.body, c.created_at
    FROM comments c
    JOIN users u ON c.user_id = u.id
    WHERE c.post_id = ?
    ORDER BY c.created_at DESC`
	rows, err := db.Query(commentQuery, postID)
	if err != nil {
		log.Printf("Error extracting comments: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
		return
	}
	defer rows.Close() // Ensure rows are closed to free resources.

	// Iterate through each row and populate the comments slice.
	for rows.Next() {
		var comment models.Comment // Temporary variable to hold comment data.
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Username, &comment.Body, &comment.CreatedAt); err != nil {
			log.Printf("Error reading comments: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
			return
		}
		comments = append(comments, comment) // Add the comment to the slice.
	}

	// Fetch like/dislike counts for post and comments
	postLikes, err := CountLikes(db, postID, "post") // Call the CountLikes function to get the number of likes for a specific post, passing the database connection, the post's ID, and the type ("post").
	if err != nil {                                  // Check if there was an error during the execution of CountLikes.
		log.Printf("Error getting amount of likes for the post: %v", err) // Log the error message, including details of the error for debugging.
	}

	postDislikes, err := CountDislikes(db, postID, "post") // Call the CountDislikes function to get the number of dislikes for the same post.
	if err != nil {                                        // Check if there was an error during the execution of CountDislikes.
		log.Printf("Error getting amount of dislikes for the post: %v", err) // Log the error message with relevant details.
	}

	// Get like/dislike counts for each comment
	commentCounts := make(map[int]models.LikeDislikeCount) // Initialize a map to store the like/dislike counts for each comment, using comment IDs as keys.
	for _, comment := range comments {                     // Loop through the list of comments for the post.
		commentLikes, err := CountLikes(db, comment.ID, "comment") // Get the number of likes for the current comment.
		if err != nil {                                            // Check if there was an error in getting the like count.
			log.Printf("Error getting amount of likes for the comments %d: %v", comment.ID, err) // Log the error with the comment ID for context.
			continue                                                                             // Skip further processing for this comment due to the error.
		}
		commentDislikes, err := CountDislikes(db, comment.ID, "comment") // Get the number of dislikes for the current comment.
		if err != nil {                                                  // Check if there was an error in getting the dislike count.
			log.Printf("Error getting amount of dislikes for the comments %d: %v", comment.ID, err) // Log the error with the comment ID for context.
			continue                                                                                // Skip further processing for this comment due to the error.
		}
		commentCounts[comment.ID] = models.LikeDislikeCount{ // Add the like and dislike counts for the current comment to the map.
			Likes:    commentLikes,    // Set the number of likes.
			Dislikes: commentDislikes, // Set the number of dislikes.
		}
	}

	// Fetch categories from the database
	rowsCategory, err := db.Query("SELECT id, name FROM categories") // Execute a SQL query to fetch all categories from the database.
	if err != nil {                                                  // Check if there was an error executing the query.
		log.Printf("Error loading categories: %v", err)                                       // Log the error with details for debugging.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories") // Render an error page with HTTP 500 status code and a descriptive message.
		return                                                                                // Exit the function to prevent further execution.
	}
	defer rowsCategory.Close() // Ensure the database rows are closed after processing, to free resources.

	var categories []models.Category // Initialize a slice to store the fetched categories.
	for rowsCategory.Next() {        // Iterate through each row in the query result.
		var category models.Category                                            // Create a variable to hold the current category data.
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil { // Read the current row's data into the category variable.
			log.Printf("Error reading categories: %v", err)                                       // Log any errors encountered during the scan.
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories") // Render an error page and exit.
			return
		}
		categories = append(categories, category) // Add the successfully read category to the categories slice.
	}

	if err := rowsCategory.Err(); err != nil { // Check for any errors that occurred during row iteration.
		log.Printf("Error parsing categories: %v", err)                                       // Log the error details for debugging.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories") // Render an error page and exit.
		return
	}

	// Add the error message to the page data
	pageData := models.PostPageData{ // Populate the pageData structure with all relevant information for rendering the page.
		Post:          post,          // The post being viewed.
		User:          user,          // The user viewing the post.
		Comments:      comments,      // The list of comments associated with the post.
		PostLikes:     postLikes,     // The total number of likes for the post.
		PostDislikes:  postDislikes,  // The total number of dislikes for the post.
		CommentCounts: commentCounts, // Like and dislike counts for each comment.
		Author:        author,        // The author of the post.
		Category:      categoryName,  // The category associated with the post.
		Categories:    categories,    // The list of all categories.
		ErrorMessage:  errorMessage,  // Any error message to display on the page.
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/post.html") // Parse the HTML templates for rendering the page.
	if err != nil {                                                                              // Check if there was an error parsing the templates.
		log.Printf("Error loading template: %v", err)                                       // Log the error details for debugging.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template") // Render an error page and exit.
		return
	}

	w.Header().Set("Content-Type", "text/html")                       // Set the response content type to HTML.
	if err := tmpl.ExecuteTemplate(w, "post", pageData); err != nil { // Render the "post" template with the provided pageData.
		log.Printf("Rendering error: %v", err)                                            // Log any errors encountered during rendering.
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page") // Render an error page and exit.
	}
}
