package models

// Importing necessary packages
import (
	"database/sql" // Provides SQL database interfaces
	"time"         // Handles time and date formatting
)

// User represents a user in the forum system
type User struct {
	ID           int       `db:"id"`            // Unique identifier for the user, corresponds to the "id" column in the database
	Username     string    `db:"username"`      // Username chosen by the user, stored in the "username" column
	PasswordHash string    `db:"password_hash"` // Hashed password for security, stored in "password_hash" column
	Email        string    `db:"email"`         // Email address of the user, stored in the "email" column
	Role         string    `db:"role"`          // User's role (e.g., admin, member), stored in "role" column
	CreatedAt    time.Time `db:"created_at"`    // Timestamp of user creation, mapped to "created_at" column
	Bio          *string   `db:"bio"`           // Optional biography, can be null, mapped to "bio" column
	ProfImage    *string   `db:"profile_image"` // Optional profile image path, can be null, stored in "profile_image" column
}

// Category represents a category of posts in the forum
type Category struct {
	ID          int            `db:"id"`          // Unique identifier for the category, corresponds to the "id" column in the database
	Name        string         `db:"name"`        // Name of the category, stored in "name" column
	Description sql.NullString `db:"description"` // Optional description of the category, supports null values, mapped to "description"
	CreatedAt   time.Time      `db:"created_at"`  // Timestamp of category creation, stored in "created_at" column
}

// Post represents a forum post
type Post struct {
	ID           int       `db:"id"`          // Unique identifier for the post, corresponds to the "id" column
	UserID       int       `db:"user_id"`     // ID of the user who created the post, mapped to "user_id" column
	Title        string    `db:"title"`       // Title of the post, stored in "title" column
	Body         string    `db:"body"`        // Content of the post, stored in "body" column
	CategoryID   int       `db:"category_id"` // ID of the category the post belongs to, mapped to "category_id"
	CreatedAt    time.Time `db:"created_at"`  // Timestamp of post creation, stored in "created_at" column
	Author       string    // Author's username, not mapped to the database
	CategoryName string    // Name of the post's category, not mapped to the database
}

// Comment represents a comment on a forum post
type Comment struct {
	ID        int       `db:"id"`         // Unique identifier for the comment, corresponds to the "id" column
	PostID    int       `db:"post_id"`    // ID of the post the comment belongs to, mapped to "post_id"
	UserID    int       `db:"user_id"`    // ID of the user who made the comment, stored in "user_id"
	Body      string    `db:"body"`       // Content of the comment, stored in "body" column
	CreatedAt time.Time `db:"created_at"` // Timestamp of comment creation, mapped to "created_at"
	Title     string    // Title of the post being commented on, not stored in the database
	Username  string    // Username of the commenter, not mapped to the database
}

// LikeDislike represents a user's reaction (like or dislike) to a post or comment
type LikeDislike struct {
	ID         int       `db:"id"`          // Unique identifier for the like/dislike, corresponds to "id"
	UserID     int       `db:"user_id"`     // ID of the user who reacted, stored in "user_id"
	TargetID   int       `db:"target_id"`   // ID of the target (post or comment), mapped to "target_id"
	TargetType string    `db:"target_type"` // Type of the target (e.g., "post" or "comment"), stored in "target_type"
	IsLike     bool      `db:"is_like"`     // True for like, false for dislike, stored in "is_like"
	CreatedAt  time.Time `db:"created_at"`  // Timestamp of the reaction, stored in "created_at"
}

// IndexPageData contains data for rendering the index page
type IndexPageData struct {
	Posts      []Post     // List of posts to display on the page
	User       *User      // Current logged-in user (if any)
	Categories []Category // List of categories for navigation
}

// PostsPageData contains data for rendering a page displaying multiple posts
type PostsPageData struct {
	Posts      []Post     // List of posts
	User       *User      // Current logged-in user
	Categories []Category // List of categories
}

// PostPageData contains data for rendering a single post page
type PostPageData struct {
	Post          Post                     // Post to display
	User          *User                    // Current logged-in user
	Comments      []Comment                // List of comments on the post
	PostLikes     int                      // Count of likes for the post
	PostDislikes  int                      // Count of dislikes for the post
	CommentCounts map[int]LikeDislikeCount // Mapping of comment IDs to their like/dislike counts
	Categories    []Category               // List of categories
	Author        string                   // Author of the post
	Category      string                   // Category name of the post
	ErrorMessage  string                   // Error message to display (if any)
}

// LikeDislikeCount holds like and dislike counts for a target
type LikeDislikeCount struct {
	Likes    int // Count of likes
	Dislikes int // Count of dislikes
}

// NewPostPageData contains data for rendering the new post creation page
type NewPostPageData struct {
	User         *User      // Current logged-in user
	Categories   []Category // List of categories for selection
	ErrorMessage string     // Error message to display (if any)
}

// LoginPageData contains data for rendering the login page
type LoginPageData struct {
	Error      string     // Error message to display (if any)
	User       *User      // Current logged-in user
	Categories []Category // List of categories
}

// ErrorPageData contains data for rendering an error page
type ErrorPageData struct {
	ErrorTitle   string     // Title of the error
	ErrorMessage string     // Error message
	User         *User      // Current logged-in user
	Categories   []Category // List of categories
}

// RegisterPageData contains data for rendering the registration page
type RegisterPageData struct {
	CaptchaQuestion string     // Captcha question for verification
	User            *User      // Current logged-in user
	Categories      []Category // List of categories
	Error           string     // Error message to display (if any)
}

// UserPageData contains data for rendering a user's profile page
type UserPageData struct {
	User       *User      // User data for the profile
	Categories []Category // List of categories
}

// UserCommentsPageData contains data for rendering a user's comments page
type UserCommentsPageData struct {
	User       *User      // Current logged-in user
	Comments   []Comment  // List of comments by the user
	Categories []Category // List of categories
}

// SearchResultsPageData contains data for rendering search results
type SearchResultsPageData struct {
	Query      string     // Search query
	Results    []Post     // List of posts matching the query
	User       *User      // Current logged-in user
	Categories []Category // List of categories
}
