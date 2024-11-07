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

	"golang.org/x/crypto/bcrypt"

	models "literary-lions/internal/models"
	"literary-lions/internal/utils"
)

func HandleRegistration(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	// Если метод GET — показываем форму регистрации с капчей
	if r.Method == http.MethodGet {
		// Генерация капчи
		captcha := utils.GenerateCaptcha()

		// Сериализация капчи в JSON
		captchaJSON, err := json.Marshal(captcha)
		if err != nil {
			log.Printf("Ошибка сериализации капчи: %v", err)
			http.Error(w, "Ошибка генерации капчи", http.StatusInternalServerError)
			return
		}

		// Кодирование JSON в Base64
		captchaBase64 := base64.StdEncoding.EncodeToString(captchaJSON)

		// Сохранение закодированного значения в cookie
		http.SetCookie(w, &http.Cookie{
			Name:   "captcha_answer",
			Value:  captchaBase64,
			Path:   "/register",
			MaxAge: 60,
		})

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
		pageData := models.RegisterPageData{
			CaptchaQuestion: captcha.Question, // Вопрос капчи
			User:            user,             // может быть nil, если пользователь не залогинен
			Categories:      categories,
		}

		// Загрузка шаблонов header и register
		tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/register.html")
		if err != nil {
			log.Printf("Ошибка загрузки шаблона: %v", err)
			http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
			return
		}

		// Установка заголовка Content-Type
		w.Header().Set("Content-Type", "text/html")

		// Рендеринг страницы регистрации с header
		err = tmpl.ExecuteTemplate(w, "register", pageData) // используем "register" для шаблона регистрации
		if err != nil {
			log.Printf("Ошибка рендеринга: %v", err)
			http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
			return
		}

		return
	}

	// Если метод POST — обработка данных формы
	if r.Method == http.MethodPost {
		captchaInput := r.FormValue("captcha")

		// Извлечение капчи из cookie
		cookie, err := r.Cookie("captcha_answer")
		if err != nil {
			http.Error(w, "Капча отсутствует или истек срок действия", http.StatusBadRequest)
			return
		}

		// Декодирование из Base64
		captchaJSON, err := base64.StdEncoding.DecodeString(cookie.Value)
		if err != nil {
			log.Printf("Ошибка декодирования капчи: %v", err)
			http.Error(w, "Ошибка обработки капчи", http.StatusBadRequest)
			return
		}

		// Десериализация JSON в структуру Captcha
		var captcha utils.Captcha
		if err := json.Unmarshal(captchaJSON, &captcha); err != nil {
			log.Printf("Ошибка десериализации капчи: %v", err)
			http.Error(w, "Ошибка обработки капчи", http.StatusBadRequest)
			return
		}

		fmt.Printf("Captcha Answer: %s, Captcha Question: %s\n", captcha.Answer, captcha.Question)

		// Проверка капчи
		if !utils.VerifyCaptcha(captchaInput, captcha) {
			http.Error(w, "Неправильный ответ на капчу", http.StatusBadRequest)
			return
		}

		// Чтение данных формы
		username := r.FormValue("username")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirmPassword")
		email := r.FormValue("email")

		// Проверка паролей
		if password != confirmPassword {
			http.Error(w, "Пароли не совпадают", http.StatusBadRequest)
			return
		}

		// Проверка на существующего пользователя с таким же именем или email
		var existingUserID int
		err = db.QueryRow("SELECT id FROM users WHERE username = ? OR email = ?", username, email).Scan(&existingUserID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Ошибка при проверке существующего пользователя: %v", err)
			http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
			return
		}

		if existingUserID != 0 {
			http.Error(w, "Пользователь с таким именем пользователя или адресом электронной почты уже существует", http.StatusConflict)
			return
		}

		// Хеширование пароля
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Ошибка хеширования пароля", http.StatusInternalServerError)
			return
		}

		// Вставка данных пользователя в базу данных
		result, err := db.Exec("INSERT INTO users (username, password_hash, email) VALUES (?, ?, ?)", username, hashedPassword, email)
		if err != nil {
			log.Printf("Ошибка при вставке пользователя: %v", err) // Логирование детали ошибки
			http.Error(w, "Ошибка базы данных при регистрации", http.StatusInternalServerError)
			return
		}

		userID, err := result.LastInsertId()
		if err != nil {
			log.Printf("Ошибка получения ID пользователя: %v", err)
			http.Error(w, "Ошибка базы данных при регистрации", http.StatusInternalServerError)
			return
		}

		// Создание токена сессии
		sessionToken, err := utils.CreateSessionToken()
		if err != nil {
			log.Printf("Ошибка при создании токена сессии: %v", err)
			http.Error(w, "Ошибка создания сессии", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec("INSERT INTO sessions (user_id, session_token, created_at) VALUES (?, ?, ?)", userID, sessionToken, time.Now())
		if err != nil {
			log.Printf("Ошибка при Создание токена сессии: %v", err)
			http.Error(w, "Ошибка создания сессии", http.StatusInternalServerError)
			return
		}

		// Сохранение токена сессии в куки
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Path:     "/",
			MaxAge:   3600, // Время жизни сессии (например, 1 час)
			Secure:   true,
			HttpOnly: true,
		})

		// Перенаправление на главную страницу после успешной регистрации
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
