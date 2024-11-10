package handlers

import (
	"database/sql"
	"html/template"
	"literary-lions/internal/models"
	"literary-lions/internal/utils"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt" // Замените на свой метод хеширования
)

func HandleLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	if r.URL.Path != "/login" {
		RenderErrorPage(w, r, db, http.StatusNotFound, "Page not found")
		return
	}

	if r.Method == http.MethodGet {
		renderLoginPage(w, r, db, "")
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username or email")
		password := r.FormValue("password")

		var user models.User
		err := db.QueryRow("SELECT id, username, email, password_hash FROM users WHERE (username = ? OR email = ?)", username, username).
			Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash)

		if err != nil {
			if err == sql.ErrNoRows {
				renderLoginPage(w, r, db, "Incorrect user's name, email or password")
			} else {
				log.Printf("Error searching user by name: %v", err)
				RenderErrorPage(w, r, db, http.StatusInternalServerError, "Database error")
			}
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			renderLoginPage(w, r, db, "Incorrect user's name, email or password")
			return
		}

		sessionToken, err := utils.CreateSessionToken()
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session token")
			return
		}

		_, err = db.Exec("INSERT INTO sessions (user_id, session_token, created_at) VALUES (?, ?, ?)", user.ID, sessionToken, time.Now())
		if err != nil {
			log.Printf("Error adding session to database: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func renderLoginPage(w http.ResponseWriter, r *http.Request, db *sql.DB, errorMessage string) {
	var user *models.User

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
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error creating session token")
			return
		}
		categories = append(categories, category)
	}

	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	pageData := models.LoginPageData{
		Error:      errorMessage,
		User:       user,
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/login.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.ExecuteTemplate(w, "login", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error rendering page")
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Удаляем сессию из базы данных
	_, err = db.Exec("DELETE FROM sessions WHERE session_token = ?", cookie.Value)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error deleting the session")
		return
	}

	// Удаляем куки с сессионным токеном
	cookie = &http.Cookie{
		Name:   "session_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)

	// Перенаправляем на главную страницу
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
