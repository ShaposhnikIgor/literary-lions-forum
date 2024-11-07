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
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Извлечение postid из пути
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 || pathParts[1] != "post" {
		http.Error(w, "Страница не найдена", http.StatusNotFound)
		return
	}

	postID, err := strconv.Atoi(pathParts[2])
	if err != nil {
		http.Error(w, "Неверный ID поста", http.StatusBadRequest)
		return
	}

	// Извлечение данных поста из базы данных
	var post models.Post
	query := "SELECT id, user_id, title, body, category_id, created_at FROM posts WHERE id = ?"
	err = db.QueryRow(query, postID).Scan(&post.ID, &post.UserID, &post.Title, &post.Body, &post.CategoryID, &post.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Пост не найден", http.StatusNotFound)
		} else {
			log.Printf("Ошибка при извлечении поста: %v", err)
			http.Error(w, "Ошибка при загрузке поста", http.StatusInternalServerError)
		}
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

	// Fetch comments for the post
	var comments []models.Comment
	commentQuery := "SELECT id, post_id, user_id, body, created_at FROM comments WHERE post_id = ? ORDER BY created_at DESC"
	rows, err := db.Query(commentQuery, postID)
	if err != nil {
		log.Printf("Ошибка при извлечении комментариев: %v", err)
		http.Error(w, "Ошибка при загрузке комментариев", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Body, &comment.CreatedAt); err != nil {
			log.Printf("Ошибка при чтении комментария1: %v", err)
			http.Error(w, "Ошибка при загрузке комментариев", http.StatusInternalServerError)
			return
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при обработке результатов комментариев: %v", err)
		http.Error(w, "Ошибка при загрузке комментариев", http.StatusInternalServerError)
		return
	}

	commentCounts := make(map[int]models.LikeDislikeCount)

	for _, comment := range comments {
		// Fetch the like and dislike counts for each comment
		commentLikes, err := CountLikes(db, comment.ID, "comment")
		if err != nil {
			log.Printf("Ошибка при получении количества лайков для комментария %d: %v", comment.ID, err)
			continue
		}
		commentDislikes, err := CountDislikes(db, comment.ID, "comment")
		if err != nil {
			log.Printf("Ошибка при получении количества дизлайков для комментария %d: %v", comment.ID, err)
			continue
		}

		// Store the counts in the map using the comment ID as the key
		commentCounts[comment.ID] = models.LikeDislikeCount{
			Likes:    commentLikes,
			Dislikes: commentDislikes,
		}
	}

	postLikes, err := CountLikes(db, postID, "post")
	if err != nil {
		log.Printf("Ошибка при получении количества лайков для поста: %v", err)
	}
	postDislikes, err := CountDislikes(db, postID, "post")
	if err != nil {
		log.Printf("Ошибка при получении количества дизлайков для поста: %v", err)
	}

	// Fetch categories from the database
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Ошибка загрузки категорий: %v", err)
		http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
		return
	}
	defer rowsCategory.Close()

	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Ошибка при чтении категории: %v", err)
			http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
			return
		}
		categories = append(categories, category)
	}

	if err := rowsCategory.Err(); err != nil {
		log.Printf("Ошибка при обработке категорий: %v", err)
		http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
		return
	}

	// Создаем структуру для передачи в шаблон
	pageData := models.PostPageData{
		Post:          post,
		User:          user, // может быть nil, если пользователь не залогинен
		Comments:      comments,
		PostLikes:     postLikes,
		PostDislikes:  postDislikes,
		CommentCounts: commentCounts,
		Categories:    categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/post.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "text/html")

	err = tmpl.ExecuteTemplate(w, "post", pageData) // specify "post" here
	if err != nil {
		log.Printf("Ошибка рендеринга: %v", err)
		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
		return
	}
}

func AllPostsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры category_id и user_id из запроса, если они существуют
	categoryIDStr := r.URL.Query().Get("category_id")
	userIDStr := r.URL.Query().Get("user_id")

	var categoryID, userID int
	var errCategory, errUser error

	// Convert category_id and user_id if present
	if categoryIDStr != "" {
		categoryID, errCategory = strconv.Atoi(categoryIDStr)
		if errCategory != nil {
			http.Error(w, "Invalid category ID", http.StatusBadRequest)
			return
		}
		// Проверка существования категории
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)", categoryID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "Категория не найдена", http.StatusNotFound)
			return
		}
	}
	if userIDStr != "" {
		userID, errUser = strconv.Atoi(userIDStr)
		if errUser != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		// Проверка существования пользователя
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userID).Scan(&exists)
		if err != nil || !exists {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
			return
		}
	}

	// Determine query based on the provided parameters
	var rows *sql.Rows
	var err error
	if categoryIDStr != "" && userIDStr != "" {
		// Fetch posts by both category and user
		rows, err = db.Query("SELECT id, user_id, title, body, category_id, created_at FROM posts WHERE category_id = ? AND user_id = ? ORDER BY created_at DESC", categoryID, userID)
	} else if categoryIDStr != "" {
		// Fetch posts by category
		rows, err = db.Query("SELECT id, user_id, title, body, category_id, created_at FROM posts WHERE category_id = ? ORDER BY created_at DESC", categoryID)
	} else if userIDStr != "" {
		// Fetch posts by user
		rows, err = db.Query("SELECT id, user_id, title, body, category_id, created_at FROM posts WHERE user_id = ? ORDER BY created_at DESC", userID)
	} else {
		// Fetch all posts if no filters are applied
		rows, err = db.Query("SELECT id, user_id, title, body, category_id, created_at FROM posts ORDER BY created_at DESC")
	}

	if err != nil {
		log.Printf("Ошибка при получении постов: %v", err)
		http.Error(w, "Ошибка при загрузке постов", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	const limit = 200
	var posts []models.Post

	// Process each row
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Body, &post.CategoryID, &post.CreatedAt); err != nil {
			log.Printf("Ошибка при чтении поста: %v", err)
			http.Error(w, "Ошибка при загрузке постов", http.StatusInternalServerError)
			return
		}
		// Truncate post body for summary display
		post.Body = truncate(post.Body, limit)
		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при обработке результата: %v", err)
		http.Error(w, "Ошибка при загрузке постов", http.StatusInternalServerError)
		return
	}

	// Проверка на наличие сессии пользователя
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var sessionUserID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&sessionUserID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", sessionUserID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Ошибка при получении пользователя: %v", err)
			}
		}
	}

	// Fetch categories from the database
	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Ошибка загрузки категорий: %v", err)
		http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
		return
	}
	defer rowsCategory.Close()

	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Ошибка при чтении категории: %v", err)
			http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
			return
		}
		categories = append(categories, category)
	}

	if err := rowsCategory.Err(); err != nil {
		log.Printf("Ошибка при обработке категорий: %v", err)
		http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
		return
	}

	// Создаем структуру для передачи в шаблон
	pageData := models.PostsPageData{
		Posts:      posts,
		User:       user,
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/all_posts.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	err = tmpl.ExecuteTemplate(w, "all_posts", pageData)
	if err != nil {
		log.Printf("Ошибка рендеринга: %v", err)
		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
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
		// Проверка на наличие сессии пользователя
		var user *models.User
		cookie, err := r.Cookie("session_token")
		if err == nil {
			var userID int
			err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
			if err == nil {
				user = &models.User{}
				err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
				if err != nil {
					log.Printf("Ошибка при получении пользователя: %v", err)
				}
			}
		} else {
			http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
			return
		}

		// Fetch categories from the database
		rows, err := db.Query("SELECT id, name FROM categories")
		if err != nil {
			log.Printf("Ошибка загрузки категорий: %v", err)
			http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var categories []models.Category
		for rows.Next() {
			var category models.Category
			if err := rows.Scan(&category.ID, &category.Name); err != nil {
				log.Printf("Ошибка при чтении категории: %v", err)
				http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
				return
			}
			categories = append(categories, category)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Ошибка при обработке категорий: %v", err)
			http.Error(w, "Ошибка загрузки категорий", http.StatusInternalServerError)
			return
		}

		// Создаем структуру для передачи в шаблон
		pageData := models.NewPostPageData{
			User:       user,
			Categories: categories,
		}

		tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/new_post.html")
		if err != nil {
			log.Printf("Ошибка загрузки шаблона: %v", err)
			http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")

		err = tmpl.ExecuteTemplate(w, "new_post", pageData)
		if err != nil {
			log.Printf("Ошибка рендеринга: %v", err)
			http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
			return
		}
		return
	}

	if r.Method == http.MethodPost {
		// Check if the user is logged in
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
			return
		}

		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err != nil {
			http.Error(w, "Ошибка при проверке сессии", http.StatusUnauthorized)
			return
		}

		// Parse form data
		err = r.ParseForm()
		if err != nil {
			http.Error(w, "Ошибка обработки формы", http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		body := r.FormValue("body")
		categoryIDStr := r.FormValue("category_id")

		if title == "" || body == "" {
			http.Error(w, "Все поля должны быть заполнены", http.StatusBadRequest)
			return
		}

		categoryID, err := strconv.Atoi(categoryIDStr)
		if err != nil {
			http.Error(w, "Неверный ID категории", http.StatusBadRequest)
			return
		}

		result, err := db.Exec("INSERT INTO posts (user_id, title, body, category_id, created_at) VALUES (?, ?, ?, ?, ?)", userID, title, body, categoryID, time.Now())
		if err != nil {
			log.Printf("Ошибка при создании поста: %v", err)
			http.Error(w, "Ошибка при создании поста", http.StatusInternalServerError)
			return
		}

		postID, err := result.LastInsertId()
		if err != nil {
			http.Error(w, "Ошибка получения ID поста", http.StatusInternalServerError)
			return
		}

		// Redirect to the new post page
		http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
	}
}
