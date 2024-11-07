package main

import (
	database "literary-lions/internal/db"
	"literary-lions/internal/handlers"
	"log"
	"net/http"
)

func main() {
	db := database.InitDB("internal/db/forum.db")
	defer db.Close()

	//db.CreateTables(db) //TBD: not needed nost likely

	// Обработчик для страницы регистрации
	// http.HandleFunc("/register", handlers.RegisterHandler)

	// Статические файлы, если есть (например, CSS)
	//http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Set up handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleIndex(w, r, db)
	})

	http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleUserPage(w, r, db)
	})

	http.HandleFunc("/user/comments", func(w http.ResponseWriter, r *http.Request) {
		handlers.UserCommentsHandler(w, r, db)
	})
	http.HandleFunc("/user/likes", func(w http.ResponseWriter, r *http.Request) {
		handlers.UserLikesHandler(w, r, db)
	})

	http.HandleFunc("/user/change_username", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleChangeUsername(w, r, db)
	})
	http.HandleFunc("/user/change_password", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleChangePassword(w, r, db)
	})
	http.HandleFunc("/user/upload_image", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleUploadProfileImage(w, r, db)
	})
	http.HandleFunc("/user/profile_image", func(w http.ResponseWriter, r *http.Request) {
		handlers.ServeProfileImage(w, r, db)
	})
	http.HandleFunc("/user/add_bio", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleChangeBio(w, r, db)
	})
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleRegistration(w, r, db)
	})

	http.HandleFunc("/post/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(w, r, db)
	})

	http.HandleFunc("/all_posts", func(w http.ResponseWriter, r *http.Request) {
		handlers.AllPostsHandler(w, r, db)
	})

	http.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		handlers.CategoriesHandler(w, r, db)
	})

	http.HandleFunc("/comment", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateCommentHandler(w, r, db)
	})

	http.HandleFunc("/new-post", func(w http.ResponseWriter, r *http.Request) {
		handlers.NewPostHandler(w, r, db)
	})

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		handlers.SearchHandler(w, r, db)
	})

	// http.HandleFunc("/post", handlers.CreatePostHandler).Methods("POST")
	// http.HandleFunc("/comment", handlers.CreateComment).Methods("POST")
	// http.HandleFunc("/posts/filter", forum.FilterPostsHandler).Methods("GET")

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleLogin(w, r, db)
	})

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handlers.LogoutHandler(w, r, db)
	})

	http.HandleFunc("/like_dislike", func(w http.ResponseWriter, r *http.Request) {
		handlers.LikeDislikeHandler(w, r, db)
	})

	//log.Fatal(http.ListenAndServe(":8080", nil))
	http.ListenAndServe(":8080", nil)
}

// Middleware для обработки ошибок
func ErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Внутренняя ошибка сервера: %v", err)
				http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
