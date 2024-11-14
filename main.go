package main

import (
	database "literary-lions/internal/db"    // Importing the database package for DB initialization and interactions
	"literary-lions/internal/handlers"      // Importing the handlers package to handle HTTP requests
	"log"                                  // Importing the log package for logging errors and information
	"net/http"                             // Importing the net/http package to work with HTTP requests and responses
)

func main() {
	// Initialize the database connection and ensure it closes when the main function exits
	db := database.InitDB("internal/db/forum.db")
	defer db.Close()  // Close the DB connection when the main function ends

	// Serve static files from the "assets/static" directory
	fs := http.FileServer(http.Dir("assets/static")) // Create a file server for static files
	http.Handle("/assets/static/", http.StripPrefix("/assets/static/", fs)) // Serve static files with URL prefix stripped

	// Main route: handles requests to the root ("/") URL, serving the index page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleIndex(w, r, db) // Call the handler for the index page
	})

	// User-specific routes for various functionalities
	http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleUserPage(w, r, db) // Display the user profile page
	})
	http.HandleFunc("/user/comments", func(w http.ResponseWriter, r *http.Request) {
		handlers.UserCommentsHandler(w, r, db) // Handle user comments
	})
	http.HandleFunc("/user/likes", func(w http.ResponseWriter, r *http.Request) {
		handlers.UserLikesHandler(w, r, db) // Handle user likes
	})
	http.HandleFunc("/user/change_username", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleChangeUsername(w, r, db) // Handle username change
	})
	http.HandleFunc("/user/change_password", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleChangePassword(w, r, db) // Handle password change
	})
	http.HandleFunc("/user/upload_image", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleUploadProfileImage(w, r, db) // Handle profile image upload
	})
	http.HandleFunc("/user/profile_image", func(w http.ResponseWriter, r *http.Request) {
		handlers.ServeProfileImage(w, r, db) // Serve the user's profile image
	})
	http.HandleFunc("/user/add_bio", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleChangeBio(w, r, db) // Handle the addition or change of user's bio
	})

	// Authentication and registration routes
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleRegistration(w, r, db) // Handle user registration
	})
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleLogin(w, r, db) // Handle user login
	})
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handlers.LogoutHandler(w, r, db) // Handle user logout
	})

	// Post and comment routes
	http.HandleFunc("/post/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(w, r, db) // Handle individual post view
	})
	http.HandleFunc("/all_posts", func(w http.ResponseWriter, r *http.Request) {
		handlers.AllPostsHandler(w, r, db) // Handle displaying all posts
	})
	http.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		handlers.CategoriesHandler(w, r, db) // Handle displaying post categories
	})
	http.HandleFunc("/comment", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateCommentHandler(w, r, db) // Handle creating comments
	})
	http.HandleFunc("/new-post", func(w http.ResponseWriter, r *http.Request) {
		handlers.NewPostHandler(w, r, db) // Handle creating new posts
	})
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		handlers.SearchHandler(w, r, db) // Handle search functionality
	})

	// Like/dislike routes for posts and comments
	http.HandleFunc("/like_dislike", func(w http.ResponseWriter, r *http.Request) {
		handlers.LikeDislikeHandler(w, r, db) // Handle liking or disliking posts
	})
	http.HandleFunc("/comment_like/", func(w http.ResponseWriter, r *http.Request) {
		handlers.CommentLikeHandler(w, r, db) // Handle liking or disliking comments
	})

	// Start the HTTP server on port 8080
	log.Println("Server started on :8080")  // Log the server start
	log.Fatal(http.ListenAndServe(":8080", nil)) // Start the server and log any fatal errors
}
