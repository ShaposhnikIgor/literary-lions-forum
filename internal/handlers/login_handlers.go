package handlers

import (
	"database/sql"
	"html/template"
	"literary-lions/internal/models"
	"literary-lions/internal/utils"
	"log"
	"net/http"
	"time"

	//"log"
	//"text/template"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt" // Замените на свой метод хеширования
)

// Обработчик входа
func HandleLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Проверяем, что метод запроса POST
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		//if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Если URL не соответствует /register
	if r.URL.Path != "/login" {
		http.Error(w, "Страница не найдена", http.StatusNotFound)
		return // Завершаем выполнение после отправки ошибки
	}

	// Для GET запроса — просто показываем форму регистрации
	if r.Method == http.MethodGet {
		// // Подключаем шаблон страницы регистрации
		// tmpl, err := template.ParseFiles("assets/template/login.html")
		// if err != nil {
		// 	log.Printf("Ошибка загрузки шаблона: %v", err)
		// 	http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		// 	return // Завершаем выполнение после отправки ошибки
		// }

		// // Устанавливаем заголовок и рендерим страницу
		// w.Header().Set("Content-Type", "text/html")
		// err = tmpl.Execute(w, nil)
		// if err != nil {
		// 	http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
		// 	return // Завершаем выполнение после отправки ошибки
		// }

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
		pageData := models.LoginPageData{
			Error:      "",
			User:       user, // может быть nil, если пользователь не залогинен
			Categories: categories,
		}

		tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/login.html")
		if err != nil {
			log.Printf("Ошибка загрузки шаблона: %v", err)
			http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
			return
		}

		// Set the content type
		w.Header().Set("Content-Type", "text/html")

		// Execute the "index" template as the main entry point
		err = tmpl.ExecuteTemplate(w, "login", pageData) // specify "index" here
		if err != nil {
			log.Printf("Ошибка рендеринга: %v", err)
			http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
			return
		}

		return // Завершаем выполнение для GET запроса
	}

	// Для POST запроса — обработка данных формы регистрации
	if r.Method == http.MethodPost {
		// Чтение данных формы
		username := r.FormValue("username or email")
		password := r.FormValue("password")

		// Поиск пользователя по имени
		var user models.User
		err := db.QueryRow("SELECT id, username, email, password_hash FROM users WHERE (username = ? OR email = ?)", username, username).
			Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Неверное имя пользователя или пароль", http.StatusUnauthorized)
			} else {
				log.Printf("Ошибка при Поиск пользователя по имени: %v", err) // Логирование детали ошибки
				http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
			}
			return
		}

		// Сравнение введенного пароля с хешем в базе данных
		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) // Замените на свой метод проверки
		if err != nil {
			http.Error(w, "Неверное имя пользователя, email или пароль", http.StatusUnauthorized)
			return
		}

		// Создание сессии (например, с использованием UUID)
		sessionToken, err := utils.CreateSessionToken()
		if err != nil {
			http.Error(w, "Ошибка создания сессионного токена", http.StatusInternalServerError)
			return
		}

		// Вставка сессии в базу данных
		_, err = db.Exec("INSERT INTO sessions (user_id, session_token, created_at) VALUES (?, ?, ?)", user.ID, sessionToken, time.Now())
		if err != nil {
			log.Printf("Ошибка при Вставка сессии в базу данных: %v", err)
			http.Error(w, "Ошибка создания сессии", http.StatusInternalServerError)
			return
		}

		// Устанавливаем cookie с сессионным токеном
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
		})

		// Перенаправление на главную страницу после успешного входа
		http.Redirect(w, r, "/", http.StatusSeeOther)
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
		http.Error(w, "Ошибка удаления сессии", http.StatusInternalServerError)
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
