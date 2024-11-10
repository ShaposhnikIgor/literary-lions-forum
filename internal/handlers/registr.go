package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"time"

	"net/http"

	models "literary-lions/internal/models"
	"literary-lions/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

func HandleRegistration(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var errorMessage string

	if r.Method == http.MethodGet {
		serveRegistrationPage(w, r, db, errorMessage)
		return
	}

	if r.Method == http.MethodPost {
		// Get CAPTCHA and form inputs
		captchaInput := r.FormValue("captcha")
		username, password, confirmPassword, email := r.FormValue("username"), r.FormValue("password"), r.FormValue("confirmPassword"), r.FormValue("email")

		// Validate CAPTCHA
		captchaValid, err := validateCaptcha(r, captchaInput)
		if err != nil {
			errorMessage = "Ошибка обработки капчи"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}
		if !captchaValid {
			errorMessage = "Неправильный ответ на капчу"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		// Check if passwords match
		if password != confirmPassword {
			errorMessage = "Пароли не совпадают"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		// Check for existing user
		var existingUserID int
		err = db.QueryRow("SELECT id FROM users WHERE username = ? OR email = ?", username, email).Scan(&existingUserID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Ошибка при проверке существующего пользователя: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка базы данных")
			return
		}
		if existingUserID != 0 {
			errorMessage = "Пользователь с таким именем пользователя или адресом электронной почты уже существует"
			serveRegistrationPage(w, r, db, errorMessage)
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка хеширования пароля")
			return
		}

		// Insert user into database
		result, err := db.Exec("INSERT INTO users (username, password_hash, email) VALUES (?, ?, ?)", username, hashedPassword, email)
		if err != nil {
			log.Printf("Ошибка при вставке пользователя: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка базы данных")
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Ошибка получения ID пользователя: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка базы данных")
			return
		}

		// Create session token
		sessionToken, err := utils.CreateSessionToken()
		if err != nil {
			log.Printf("Ошибка при создании токена сессии: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка создания сессии")
			return
		}

		_, err = db.Exec("INSERT INTO sessions (user_id, session_token, created_at) VALUES (?, ?, ?)", userID, sessionToken, time.Now())
		if err != nil {
			log.Printf("Ошибка при создании токена сессии: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка создания сессии")
			return
		}

		// Set session token cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Path:     "/",
			MaxAge:   3600,
			Secure:   true,
			HttpOnly: true,
		})

		// Redirect to home page after successful registration
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func serveRegistrationPage(w http.ResponseWriter, r *http.Request, db *sql.DB, errorMessage string) {
	captcha := utils.GenerateCaptcha()

	captchaJSON, err := json.Marshal(captcha)
	if err != nil {
		log.Printf("Ошибка сериализации капчи: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка генерации капчи")
		return
	}

	captchaBase64 := base64.StdEncoding.EncodeToString(captchaJSON)
	http.SetCookie(w, &http.Cookie{
		Name:   "captcha_answer",
		Value:  captchaBase64,
		Path:   "/register",
		MaxAge: 60,
	})

	var user *models.User
	if sessionCookie, err := r.Cookie("session_token"); err == nil {
		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", sessionCookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
			if err != nil {
				log.Printf("Ошибка при получении пользователя: %v", err)
			}
		}
	}

	rowsCategory, err := db.Query("SELECT id, name FROM categories")
	if err != nil {
		log.Printf("Ошибка загрузки категорий: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки категорий")
		return
	}
	defer rowsCategory.Close()

	var categories []models.Category
	for rowsCategory.Next() {
		var category models.Category
		if err := rowsCategory.Scan(&category.ID, &category.Name); err != nil {
			log.Printf("Ошибка при чтении категории: %v", err)
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки категорий")
			return
		}
		categories = append(categories, category)
	}

	pageData := models.RegisterPageData{
		CaptchaQuestion: captcha.Question,
		User:            user,
		Categories:      categories,
		Error:           errorMessage,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/register.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки шаблона")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err = tmpl.ExecuteTemplate(w, "register", pageData); err != nil {
		log.Printf("Ошибка рендеринга: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка рендеринга страницы")
	}
}

func validateCaptcha(r *http.Request, captchaInput string) (bool, error) {
	cookie, err := r.Cookie("captcha_answer")
	if err != nil {
		return false, fmt.Errorf("капча отсутствует или истек срок действия")
	}

	captchaJSON, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Printf("Ошибка декодирования капчи: %v", err)
		return false, fmt.Errorf("ошибка декодирования капчи")
	}

	var captcha utils.Captcha
	if err := json.Unmarshal(captchaJSON, &captcha); err != nil {
		log.Printf("Ошибка десериализации капчи: %v", err)
		return false, fmt.Errorf("oшибка десериализации капчи")
	}

	return utils.VerifyCaptcha(captchaInput, captcha), nil
}
