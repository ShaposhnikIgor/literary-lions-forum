package main

// Import necessary packages
import (
	database "literary-lions/internal/db" // Custom package for database operations
	"literary-lions/internal/handlers"    // Custom package for HTTP request handlers
	"log"                                 // For logging server messages
	"net/http"                            // Core HTTP package for handling requests
)

func main() {
	// Initialize the database connection using the InitDB function from the database package.
	// "internal/db/forum.db" is the SQLite database file.
	db := database.InitDB("internal/db/forum.db")
	// Ensure the database connection is closed when the program terminates.
	defer db.Close()

	// Serve static files, such as CSS, JS, and images, from the "assets/static" directory.
	// http.FileServer creates a handler to serve these files.
	fs := http.FileServer(http.Dir("assets/static"))
	// Register the static file handler at the "/assets/static/" route, stripping the prefix for proper file paths.
	http.Handle("/assets/static/", http.StripPrefix("/assets/static/", fs))

	// Set up a route for the main index page.
	// When the root URL ("/") is accessed, HandleIndex is called with the database passed as a parameter.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleIndex(w, r, db)
	})

	// Define routes for user-related actions.

	// Serve the user's profile page.
	http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleUserPage(w, r, db)
	})

	// Serve the user's comments page.
	http.HandleFunc("/user/comments", func(w http.ResponseWriter, r *http.Request) {
		handlers.UserCommentsHandler(w, r, db)
	})

	// Serve the user's likes page.
	http.HandleFunc("/user/likes", func(w http.ResponseWriter, r *http.Request) {
		handlers.UserLikesHandler(w, r, db)
	})

	// Allow the user to change their username.
	http.HandleFunc("/user/change_username", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleChangeUsername(w, r, db)
	})

	// Allow the user to change their password.
	http.HandleFunc("/user/change_password", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleChangePassword(w, r, db)
	})

	// Allow the user to upload a profile image.
	http.HandleFunc("/user/upload_image", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleUploadProfileImage(w, r, db)
	})

	// Serve the user's profile image.
	http.HandleFunc("/user/profile_image", func(w http.ResponseWriter, r *http.Request) {
		handlers.ServeProfileImage(w, r, db)
	})

	// Allow the user to add or update their bio.
	http.HandleFunc("/user/add_bio", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleChangeBio(w, r, db)
	})

	// Define routes for authentication and registration.

	// Handle user registration requests.
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleRegistration(w, r, db)
	})

	// Handle user login requests.
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleLogin(w, r, db)
	})

	// Handle user logout requests.
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handlers.LogoutHandler(w, r, db)
	})

	// Define routes for post and comment-related actions.

	// Serve individual posts based on the URL pattern.
	http.HandleFunc("/post/", func(w http.ResponseWriter, r *http.Request) {
		handlers.PostHandler(w, r, db)
	})

	// Serve all posts on a dedicated page.
	http.HandleFunc("/all_posts", func(w http.ResponseWriter, r *http.Request) {
		handlers.AllPostsHandler(w, r, db)
	})

	// Serve the categories page.
	http.HandleFunc("/categories", func(w http.ResponseWriter, r *http.Request) {
		handlers.CategoriesHandler(w, r, db)
	})

	// Handle requests to create a new comment on a post.
	http.HandleFunc("/comment", func(w http.ResponseWriter, r *http.Request) {
		handlers.CreateCommentHandler(w, r, db)
	})

	// Handle requests to create a new post.
	http.HandleFunc("/new-post", func(w http.ResponseWriter, r *http.Request) {
		handlers.NewPostHandler(w, r, db)
	})

	// Handle search queries.
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		handlers.SearchHandler(w, r, db)
	})

	// Define routes for liking/disliking posts or comments.

	// Handle likes/dislikes for posts.
	http.HandleFunc("/like_dislike", func(w http.ResponseWriter, r *http.Request) {
		handlers.LikeDislikeHandler(w, r, db)
	})

	// Handle likes for individual comments.
	http.HandleFunc("/comment_like/", func(w http.ResponseWriter, r *http.Request) {
		handlers.CommentLikeHandler(w, r, db)
	})

	// Start the HTTP server on port 8080.
	// Log a message indicating the server has started.
	log.Println("Server started on :8080")
	// Use log.Fatal to log any errors encountered by the server and terminate the program if needed.
	log.Fatal(http.ListenAndServe(":8080", nil))
}
