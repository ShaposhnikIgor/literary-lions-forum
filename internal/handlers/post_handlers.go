package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"literary-lions/internal/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func PostHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Ensure request method is GET or POST, otherwise return Method Not Allowed
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Extract post ID from the URL path and validate
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 || pathParts[1] != "post" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page not found")
		return
	}

	// Convert post ID to an integer, return Bad Request if conversion fails
	postID, err := strconv.Atoi(pathParts[2])
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect ID of the post")
		return
	}

	// Query to retrieve post details, author, and category
	var author, categoryName string
	var post models.Post
	query := `
		SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.id = ?`
	err = db.QueryRow(query, postID).Scan(
		&post.ID, &post.UserID, &author, &post.Title, &post.Body,
		&post.CategoryID, &categoryName, &post.CreatedAt,
	)
	if err != nil {
		// Return Not Found if post does not exist, Internal Server Error otherwise
		if err == sql.ErrNoRows {
			RenderErrorPage(w, r, db, http.StatusNotFound, "Post not found")
		} else {
			log.Printf("Error extracting the post: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading post")
		}
		return
	}

	// Check if the user is logged in by looking for a session cookie
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			// Retrieve logged-in user details
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Handle like/dislike actions if request method is POST
	if r.Method == http.MethodPost {
		targetType := r.FormValue("target_type")
		isLike := r.FormValue("is_like") == "true"

		// Process like/dislike for post if target is "post"
		if targetType == "post" {
			_, err = db.Exec(`
			INSERT OR REPLACE INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
			VALUES (?, ?, ?, ?, ?)`,
				user.ID, postID, "post", isLike, time.Now())
			if err != nil {
				log.Printf("Error when adding/updating like/dislike for the post: %v", err)
			}
		}

		// Process like/dislike for comment if target is "comment"
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

		// Redirect to the same post page to refresh like/dislike counts
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
		return
	}

	// Query to retrieve comments for the post along with their authors
	var comments []models.Comment
	commentQuery := `
	SELECT c.id, c.post_id, c.user_id, u.username, c.body, c.created_at
	FROM comments c
	JOIN users u ON c.user_id = u.id
	WHERE c.post_id = ?
	ORDER BY c.created_at DESC
	`
	rows, err := db.Query(commentQuery, postID)
	if err != nil {
		log.Printf("Error extracting comments: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
		return
	}
	defer rows.Close()

	// Collect each comment in a slice
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Username, &comment.Body, &comment.CreatedAt); err != nil {
			log.Printf("Error reading comments: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
			return
		}
		comments = append(comments, comment)
	}

	// Fetch like/dislike counts for the post
	postLikes, err := CountLikes(db, postID, "post")
	if err != nil {
		log.Printf("Error getting amount of likes for the post: %v", err)
	}
	postDislikes, err := CountDislikes(db, postID, "post")
	if err != nil {
		log.Printf("Error getting amount of dislikes for the post: %v", err)
	}

	// Get like/dislike counts for each comment
	commentCounts := make(map[int]models.LikeDislikeCount)
	for _, comment := range comments {
		commentLikes, err := CountLikes(db, comment.ID, "comment")
		if err != nil {
			log.Printf("Error getting amount of likes for the comments %d: %v", comment.ID, err)
			continue
		}
		commentDislikes, err := CountDislikes(db, comment.ID, "comment")
		if err != nil {
			log.Printf("Error getting amount of dislikes for the comments %d: %v", comment.ID, err)
			continue
		}
		commentCounts[comment.ID] = models.LikeDislikeCount{
			Likes:    commentLikes,
			Dislikes: commentDislikes,
		}
	}

	// Query to fetch all categories from the database
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close()

	// Collect each category in a slice
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

	// Check for errors after processing category rows
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare data for rendering the post page
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
	}

	// Parse and render the HTML template for the post page
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/post.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set content type to HTML and execute the template with provided data
	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "post", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page")
		return
	}
}

func AllPostsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Check if the request method is GET; if not, return a 405 Method Not Allowed error.
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Mathod is not supported")
		return
	}

	// Retrieve optional category and user ID filters from the query parameters.
	categoryIDStr := r.URL.Query().Get("category_id")
	userIDStr := r.URL.Query().Get("user_id")

	var categoryID, userID int
	var errCategory, errUser error

	// Convert category_id and user_id strings to integers if present.
	if categoryIDStr != "" {
		categoryID, errCategory = strconv.Atoi(categoryIDStr)
		// If the category ID is invalid, return a 400 Bad Request error.
		if errCategory != nil {
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid category ID")
			return
		}

		// Verify if the provided category exists in the database.
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)", categoryID).Scan(&exists)
		if err != nil || !exists {
			RenderErrorPage(w, r, db, http.StatusNotFound, "Category not found")
			return
		}
	}
	if userIDStr != "" {
		userID, errUser = strconv.Atoi(userIDStr)
		// If the user ID is invalid, return a 400 Bad Request error.
		if errUser != nil {
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid user ID")
			return
		}

		// Verify if the provided user exists in the database.
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
		if err != nil || !exists {
			RenderErrorPage(w, r, db, http.StatusNotFound, "User is not found")
			return
		}
	}

	// Determine the query to fetch posts based on the provided parameters.
	var rows *sql.Rows
	var err error

	if categoryIDStr != "" && userIDStr != "" {
		// Fetch posts filtered by both category and user, including username and category name.
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			WHERE p.category_id = ? AND p.user_id = ?
			ORDER BY p.created_at DESC
		`, categoryID, userID)
	} else if categoryIDStr != "" {
		// Fetch posts filtered by category, including username and category name.
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			WHERE p.category_id = ?
			ORDER BY p.created_at DESC
		`, categoryID)
	} else if userIDStr != "" {
		// Fetch posts filtered by user, including username and category name.
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			WHERE p.user_id = ?
			ORDER BY p.created_at DESC
		`, userID)
	} else {
		// Fetch all posts without any filters, including username and category name.
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			ORDER BY p.created_at DESC
		`)
	}

	// Handle any error that occurs while querying the database.
	if err != nil {
		log.Printf("Error getting posts: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading posts")
		return
	}
	defer rows.Close()

	// Set a character limit for truncating post bodies in the list view.
	const limit = 200
	var posts []models.Post

	// Process each row in the result to build the posts slice.
	for rows.Next() {
		var post models.Post
		var author, categoryName string
		// Extract post data from the row.
		if err := rows.Scan(
			&post.ID, &post.UserID, &author, &post.Title, &post.Body, &post.CategoryID, &categoryName, &post.CreatedAt,
		); err != nil {
			log.Printf("Error extracting post's data: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error extracting post's data")
			return
		}

		// Truncate the post body for summary display and assign author and category name.
		post.Body = truncate(post.Body, limit)
		post.Author = author
		post.CategoryName = categoryName
		posts = append(posts, post)
	}

	// Check if there were any errors during row iteration.
	if err := rows.Err(); err != nil {
		log.Printf("Error parsing result: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading post")
		return
	}

	// Retrieve the logged-in user if a session cookie is present.
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var sessionUserID int
		// Fetch the user ID from the session token.
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&sessionUserID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", sessionUserID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

	// Load available categories from the database for filtering.
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Error loading categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}
	defer rowsCategory.Close()

	var categories []models.Category
	// Build the categories slice from the query results.
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Error reading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category)
	}

	// Check for any errors that occurred while reading category rows.
	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Prepare data to render on the template.
	pageData := models.PostsPageData{
		Posts:      posts,
		User:       user,
		Categories: categories,
	}

	// Load and parse the template files for rendering.
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/all_posts.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set response content type to HTML and render the page.
	w.Header().Set("Content-Type", "text/html")

	err = tmpl.ExecuteTemplate(w, "all_posts", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}

// Truncate function to limit the string length
func truncate(text string, limit int) string {
	// Check if the text length exceeds the limit
	if len(text) > limit {
		// Return truncated text with ellipsis
		return text[:limit] + "..."
	}
	// Return original text if within limit
	return text
}

func NewPostHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Handle GET request for displaying the new post form
	if r.Method == http.MethodGet {

		var user *models.User
		// Attempt to retrieve session token from cookies
		cookie, err := r.Cookie("session_token")
		if err == nil {
			var userID int
			// Validate session token and retrieve user ID
			err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
			if err == nil {
				user = &models.User{}
				// Retrieve user details based on user ID
				err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
				if err != nil {
					log.Printf("Error getting the user: %v", err)
				}
			}
		} else {
			// Display unauthorized error if session token is missing
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
			return
		}

		// Fetch categories from the database
		rows, err := db.Query("SELECT id, name FROM categories")
		if err != nil {
			log.Printf("Error loading categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		defer rows.Close()

		var categories []models.Category
		// Iterate over categories and store in the slice
		for rows.Next() {
			var category models.Category
			if err := rows.Scan(&category.ID, &category.Name); err != nil {
				log.Printf("Error reading categories: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
				return
			}
			categories = append(categories, category)
		}

		// Check for errors encountered during rows iteration
		if err := rows.Err(); err != nil {
			log.Printf("Error parsing categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}

		// Prepare data for the template
		pageData := models.NewPostPageData{
			User:       user,
			Categories: categories,
		}

		// Parse and execute HTML template
		tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/new_post.html")
		if err != nil {
			log.Printf("Error loading template: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
			return
		}

		w.Header().Set("Content-Type", "text/html")

		// Render the new post page with data
		err = tmpl.ExecuteTemplate(w, "new_post", pageData)
		if err != nil {
			log.Printf("Rendering error: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
			return
		}
		return
	}

	// Handle POST request for creating a new post
	if r.Method == http.MethodPost {
		// Check if the user is logged in by validating the session token
		cookie, err := r.Cookie("session_token")
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
			return
		}

		var userID int
		// Retrieve user ID associated with the session token
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
			return
		}

		// Parse form data from the request
		err = r.ParseForm()
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Error parsing the form")
			return
		}

		// Retrieve title, body, and category ID from form data
		title := r.FormValue("title")
		body := r.FormValue("body")
		categoryIDStr := r.FormValue("category_id")

		// Validate that title and body are not empty
		if title == "" || body == "" {
			RenderErrorPage(w, r, db, http.StatusNotFound, "All fields should be filled")
			return
		}

		// Convert category ID to integer
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusNotFound, "Incorrect ID of category")
			return
		}

		// Insert new post into the database
		result, err := db.Exec("INSERT INTO posts (user_id, title, body, category_id, created_at) VALUES (?, ?, ?, ?, ?)", userID, title, body, categoryID, time.Now())
		if err != nil {
			log.Printf("Error creating the post: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating the post")
			return
		}

		// Retrieve ID of the newly created post
		postID, err := result.LastInsertId()
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error getting ID of post")
			return
		}

		// Redirect to the new post's page
		http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
	}
}
