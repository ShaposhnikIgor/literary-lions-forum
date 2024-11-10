package models

import (
	"database/sql"
	"time"
)

// User - структура для представления пользователей
type User struct {
	ID           int       `db:"id"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	Email        string    `db:"email"`
	Role         string    `db:"role"`
	CreatedAt    time.Time `db:"created_at"`
	Bio          *string   `db:"bio"`
	ProfImage    *string   `db:"profile_image"`
	// Salt    	 string			`db:"salt"'
}

// Category - структура для представления категорий
type Category struct {
	ID          int            `db:"id"`
	Name        string         `db:"name"`
	Description sql.NullString `db:"description"` // так как поле может быть NULL
	CreatedAt   time.Time      `db:"created_at"`
}

// Post - структура для представления постов
type Post struct {
	ID           int       `db:"id"`
	UserID       int       `db:"user_id"`
	Title        string    `db:"title"`
	Body         string    `db:"body"`
	CategoryID   int       `db:"category_id"`
	CreatedAt    time.Time `db:"created_at"`
	Author       string
	CategoryName string
}

// Comment - структура для представления комментариев
type Comment struct {
	ID        int       `db:"id"`
	PostID    int       `db:"post_id"`
	UserID    int       `db:"user_id"`
	Body      string    `db:"body"`
	CreatedAt time.Time `db:"created_at"`
	Title     string
	Username  string
}

// LikeDislike - структура для представления лайков и дизлайков
type LikeDislike struct {
	ID         int       `db:"id"`
	UserID     int       `db:"user_id"`
	TargetID   int       `db:"target_id"`
	TargetType string    `db:"target_type"` // 'post' или 'comment'
	IsLike     bool      `db:"is_like"`     // TRUE для лайка, FALSE для дизлайка
	CreatedAt  time.Time `db:"created_at"`
}

type IndexPageData struct {
	Posts      []Post
	User       *User // Используем указатель, чтобы передавать nil, если пользователь не залогинен
	Categories []Category
}

type PostsPageData struct {
	Posts      []Post
	User       *User // Используем указатель, чтобы передавать nil, если пользователь не залогинен
	Categories []Category
}

type PostPageData struct {
	Post          Post
	User          *User
	Comments      []Comment
	PostLikes     int
	PostDislikes  int
	CommentCounts map[int]LikeDislikeCount
	Categories    []Category
	Author        string
	Category      string
}

type LikeDislikeCount struct {
	Likes    int
	Dislikes int
}

type NewPostPageData struct {
	User       *User // Используем указатель, чтобы передавать nil, если пользователь не залогинен
	Categories []Category
}

type LoginPageData struct {
	Error      string
	User       *User // Используем указатель, чтобы передавать nil, если пользователь не залогинен
	Categories []Category
}

type RegisterPageData struct {
	CaptchaQuestion string // Вопрос для отображения капчи
	User            *User  // Данные пользователя, могут быть nil, если пользователь не залогинен
	Categories      []Category
	Error           string
}

type UserPageData struct {
	User       *User // Данные пользователя, могут быть nil, если пользователь не залогинен
	Categories []Category
}

type UserCommentsPageData struct {
	User       *User
	Comments   []Comment
	Categories []Category
}

type SearchResultsPageData struct {
	Query      string
	Results    []Post
	User       *User
	Categories []Category
}
