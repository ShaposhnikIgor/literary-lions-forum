package main

import (
	database "literary-lions/internal/db"
	"literary-lions/internal/handlers"
	"log"
	"net/http"
)

func main() {
	// Initialize the database and ensure it closes when main exits
	db := database.InitDB("internal/db/forum.db")
	defer db.Close()

	// Serve static files from "assets/static" directory
	fs := http.FileServer(http.Dir("assets/static"))
	http.Handle("/assets/static/", http.StripPrefix("/assets/static/", fs))

	// Set up main route for the index page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleIndex(w, r, db)
	})

	// User-specific routes
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

	// Authentication and registration routes
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleRegistration(w, r, db)
	})
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleLogin(w, r, db)
	})
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handlers.LogoutHandler(w, r, db)
	})

	// Post and comment routes
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

	// Like/dislike routes for posts and comments
	http.HandleFunc("/like_dislike", func(w http.ResponseWriter, r *http.Request) {
		handlers.LikeDislikeHandler(w, r, db)
	})
	http.HandleFunc("/comment_like/", func(w http.ResponseWriter, r *http.Request) {
		handlers.CommentLikeHandler(w, r, db)
	})

	// Start the HTTP server on port 8080
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
