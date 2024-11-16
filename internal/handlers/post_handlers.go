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
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Add errorMessage to pageData in the existing logic

	// Extract post ID from the URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 || pathParts[1] != "post" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page not found")
		return
	}

	postID, err := strconv.Atoi(pathParts[2])
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect ID of the post")
		return
	}

	var author string
	var categoryName string
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
		if err == sql.ErrNoRows {
			RenderErrorPage(w, r, db, http.StatusNotFound, "Post not found")
		} else {
			log.Printf("Error extracting the post: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading post")
		}
		return
	}

	queryURL := r.URL.Query()
	errorMessage := queryURL.Get("error")

	// Check if user is logged in (session)
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

	// Handling like/dislike for the post
	if r.Method == http.MethodPost {
		targetType := r.FormValue("target_type")
		isLike := r.FormValue("is_like") == "true"

		body := strings.TrimSpace(r.FormValue("body"))
		if body == "" {
			RenderPostWithError(w, r, db, postID, "Comment cannot be empty")
			return
		}

		// Validate and process like/dislike for post
		if targetType == "post" {
			_, err = db.Exec(`
			INSERT OR REPLACE INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
			VALUES (?, ?, ?, ?, ?)`,
				user.ID, postID, "post", isLike, time.Now())
			if err != nil {
				log.Printf("Error when adding/updating like/dislike for the post: %v", err)
			}
		}

		// Handling like/dislike for comment (if any)
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

		// Redirect back to the same post page to reload and show updated counts
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
		return
	}

	// Fetch comments for the post along with usernames
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

	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Username, &comment.Body, &comment.CreatedAt); err != nil {
			log.Printf("Error reading comments: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
			return
		}
		comments = append(comments, comment)
	}

	// Fetch like/dislike counts for post and comments
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
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Render page with updated like/dislike counts
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

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/post.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "post", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page")
		return
	}
}

func AllPostsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Mathod is not supported")
		return
	}

	categoryIDStr := r.URL.Query().Get("category_id")
	userIDStr := r.URL.Query().Get("user_id")

	var categoryID, userID int
	var errCategory, errUser error

	// Convert category_id and user_id if present
	if categoryIDStr != "" {
		categoryID, errCategory = strconv.Atoi(categoryIDStr)
		if errCategory != nil {
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid category ID")
			return
		}

		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)", categoryID).Scan(&exists)
		if err != nil || !exists {
			RenderErrorPage(w, r, db, http.StatusNotFound, "Category not found")
			return
		}
	}
	if userIDStr != "" {
		userID, errUser = strconv.Atoi(userIDStr)
		if errUser != nil {
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Invalid user ID")
			return
		}

		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
		if err != nil || !exists {
			RenderErrorPage(w, r, db, http.StatusNotFound, "User is not found")
			return
		}
	}

	// Determine query based on the provided parameters
	var rows *sql.Rows
	var err error

	if categoryIDStr != "" && userIDStr != "" {
		// Fetch posts by both category and user, including the username and category name
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			WHERE p.category_id = ? AND p.user_id = ?
			ORDER BY p.created_at DESC
		`, categoryID, userID)
	} else if categoryIDStr != "" {
		// Fetch posts by category, including the username and category name
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			WHERE p.category_id = ?
			ORDER BY p.created_at DESC
		`, categoryID)
	} else if userIDStr != "" {
		// Fetch posts by user, including the username and category name
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			WHERE p.user_id = ?
			ORDER BY p.created_at DESC
		`, userID)
	} else {
		// Fetch all posts, including the username and category name
		rows, err = db.Query(`
			SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
			FROM posts p
			JOIN users u ON p.user_id = u.id
			JOIN categories c ON p.category_id = c.id
			ORDER BY p.created_at DESC
		`)
	}

	if err != nil {
		log.Printf("Error getting posts: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading posts")
		return
	}
	defer rows.Close()

	const limit = 200
	var posts []models.Post

	for rows.Next() {
		var post models.Post
		var author, categoryName string
		if err := rows.Scan(
			&post.ID, &post.UserID, &author, &post.Title, &post.Body, &post.CategoryID, &categoryName, &post.CreatedAt,
		); err != nil {
			log.Printf("Error extracting post's data: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error extracting post's data")
			return
		}

		// Truncate post body for summary display
		post.Body = truncate(post.Body, limit)
		post.Author = author
		post.CategoryName = categoryName
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error parsing result: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading post")
		return
	}

	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var sessionUserID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&sessionUserID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", sessionUserID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
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
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	pageData := models.PostsPageData{
		Posts:      posts,
		User:       user,
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/all_posts.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

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
	if len(text) > limit {
		return text[:limit] + "..."
	}
	return text
}

func NewPostHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == http.MethodGet {

		var user *models.User
		cookie, err := r.Cookie("session_token")
		if err == nil {
			var userID int
			err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
			if err == nil {
				user = &models.User{}
				err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
				if err != nil {
					log.Printf("Error getting the user: %v", err)
				}
			}
		} else {
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
		for rows.Next() {
			var category models.Category
			if err := rows.Scan(&category.ID, &category.Name); err != nil {
				log.Printf("Error reading categories: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
				return
			}
			categories = append(categories, category)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Error parsing categories: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}

		pageData := models.NewPostPageData{
			User:       user,
			Categories: categories,
		}

		tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/new_post.html")
		if err != nil {
			log.Printf("Error loading template: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
			return
		}

		w.Header().Set("Content-Type", "text/html")

		err = tmpl.ExecuteTemplate(w, "new_post", pageData)
		if err != nil {
			log.Printf("Rendering error: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
			return
		}
		return
	}

	if r.Method == http.MethodPost {
		// Check if the user is logged in
		cookie, err := r.Cookie("session_token")
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
			return
		}

		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
			return
		}

		// Parse form data
		err = r.ParseForm()
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusBadRequest, "Error parsing the form")
			return
		}

		// title := r.FormValue("title")
		// body := r.FormValue("body")
		title := strings.TrimSpace(r.FormValue("title"))
		body := strings.TrimSpace(r.FormValue("body"))
		categoryIDStr := r.FormValue("category_id")

		// if title == "" || body == "" {
		// 	RenderErrorPage(w, r, db, http.StatusNotFound, "All fields should be filled")
		// 	return
		// }

		// Validation
		if title == "" || body == "" {
			// Reload the page with an error message

			var user *models.User
			cookie, err := r.Cookie("session_token")
			if err == nil {
				var userID int
				err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
				if err == nil {
					user = &models.User{}
					err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
					if err != nil {
						log.Printf("Error getting the user: %v", err)
					}
				}
			} else {
				RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
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
				log.Printf("Error parsing categories: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
				return
			}

			pageData := models.NewPostPageData{
				User:         user,
				ErrorMessage: "All fields are required and cannot be empty.",
				Categories:   categories,
			}

			tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/new_post.html")
			if err != nil {
				log.Printf("Error loading template: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
				return
			}

			w.Header().Set("Content-Type", "text/html")
			//fmt.Println(pageData)
			tmpl.ExecuteTemplate(w, "new_post", pageData)
			return
		}

		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusNotFound, "Incorrect ID of category")
			return
		}

		result, err := db.Exec("INSERT INTO posts (user_id, title, body, category_id, created_at) VALUES (?, ?, ?, ?, ?)", userID, title, body, categoryID, time.Now())
		if err != nil {
			log.Printf("Error creating the post: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating the post")
			return
		}

		postID, err := result.LastInsertId()
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error getting ID of post")
			return
		}

		// Redirect to the new post page
		http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
	}
}

func RenderPostWithError(w http.ResponseWriter, r *http.Request, db *sql.DB, postID int, errorMessage string) {
	// postID, err := strconv.Atoi(pathParts[2])
	// if err != nil {
	// 	RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect ID of the post")
	// 	return
	// }

	var author string
	var categoryName string
	var post models.Post
	query := `
		SELECT p.id, p.user_id, u.username, p.title, p.body, p.category_id, c.name AS category_name, p.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		WHERE p.id = ?`
	err := db.QueryRow(query, postID).Scan(
		&post.ID, &post.UserID, &author, &post.Title, &post.Body,
		&post.CategoryID, &categoryName, &post.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			RenderErrorPage(w, r, db, http.StatusNotFound, "Post not found")
		} else {
			log.Printf("Error extracting the post: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading post")
		}
		return
	}

	// Check if user is logged in (session)
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

	// Handling like/dislike for the post
	if r.Method == http.MethodPost {
		targetType := r.FormValue("target_type")
		isLike := r.FormValue("is_like") == "true"

		body := strings.TrimSpace(r.FormValue("body"))
		if body == "" {
			RenderPostWithError(w, r, db, postID, "Comment cannot be empty")
			return
		}

		// Validate and process like/dislike for post
		if targetType == "post" {
			_, err = db.Exec(`
			INSERT OR REPLACE INTO likes_dislikes (user_id, target_id, target_type, is_like, created_at)
			VALUES (?, ?, ?, ?, ?)`,
				user.ID, postID, "post", isLike, time.Now())
			if err != nil {
				log.Printf("Error when adding/updating like/dislike for the post: %v", err)
			}
		}

		// Handling like/dislike for comment (if any)
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

		// Redirect back to the same post page to reload and show updated counts
		http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
		return
	}

	// Fetch comments for the post along with usernames
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

	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Username, &comment.Body, &comment.CreatedAt); err != nil {
			log.Printf("Error reading comments: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading comments")
			return
		}
		comments = append(comments, comment)
	}

	// Fetch like/dislike counts for post and comments
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
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	// Add the error message to the page data
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

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/post.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.ExecuteTemplate(w, "post", pageData); err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page")
	}
}
