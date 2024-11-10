package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	models "literary-lions/internal/models"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
)

func GetUserIDFromSession(r *http.Request, db *sql.DB) (int, error) {
	// Получаем куки с токеном сессии
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0, err // Ошибка: куки не найдены
	}

	// Извлекаем user_id по токену сессии из базы данных
	var userID int
	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
	if err != nil {
		return 0, err // Ошибка: токен не найден в сессиях
	}

	return userID, nil
}

func HandleUserPage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method is not supported")
		return
	}

	// Проверка на наличие сессии пользователя
	var user *models.User
	cookie, err := r.Cookie("session_token")
	if err == nil {
		var userID int
		err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
		if err == nil {
			user = &models.User{}
			err = db.QueryRow("SELECT id, username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username, &user.Email, &user.Bio, &user.ProfImage)
			if err != nil {
				log.Printf("Error getting the user: %v", err)
			}
		}
	}

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
			RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
			return
		}
		categories = append(categories, category)
	}

	if err := rowsCategory.Err(); err != nil {
		log.Printf("Error parsing categories: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading categories")
		return
	}

	//Создаем структуру для передачи в шаблон
	pageData := models.UserPageData{
		User:       user, // может быть nil, если пользователь не залогинен
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user.html")
	if err != nil {
		log.Printf("Error loading template: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading template")
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "text/html")

	// Execute the "index" template as the main entry point
	err = tmpl.ExecuteTemplate(w, "user", pageData)
	if err != nil {
		log.Printf("Rendering error: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Rendering page error")
		return
	}
}

func HandleChangeUsername(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession HandleChangeUsername: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	newUsername := r.FormValue("username")

	_, err = db.Exec("UPDATE users SET username = ? WHERE id = ?", newUsername, userID)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func HandleChangePassword(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Получение ID пользователя из сессии
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession в HandleChangePassword: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Чтение и проверка текущего пароля и нового пароля из формы
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if newPassword != confirmPassword {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Passwords don't match")
		return
	}

	// Извлечение хеша текущего пароля из базы данных для проверки
	var passwordHash string
	err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		log.Printf("Error getting password_hash: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Проверка текущего пароля
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword))
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusBadRequest, "Incorrect current password")
		return
	}

	// Хеширование нового пароля
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Обновление пароля в базе данных
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHashedPassword, userID)
	if err != nil {
		log.Printf("Error updating password: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Перенаправление на страницу профиля после успешного обновления пароля
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func ServeProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Извлечение пути к изображению из базы данных
	var filePath string
	err = db.QueryRow("SELECT COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&filePath)
	if err != nil {
		log.Printf("Error getting the picture path: %v", err)
		RenderErrorPage(w, r, db, http.StatusNotFound, "Picture is not found")
		return
	}

	// Определение типа изображения по расширению файла
	fileExt := filepath.Ext(filePath)
	switch fileExt {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	default:
		RenderErrorPage(w, r, db, http.StatusUnsupportedMediaType, "Format of the image is not supported")
		return
	}

	// Отправка файла
	http.ServeFile(w, r, filePath)
}

func HandleUploadProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	// Получение ID пользователя из сессии
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error getting ID of user from the session: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	// Получение текущего пути к изображению профиля из базы данных
	var oldFilePath string
	err = db.QueryRow("SELECT profile_image FROM users WHERE id = ?", userID).Scan(&oldFilePath)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error getting the path to an old image: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Error loading profile")
		return
	}

	// If no profile image is set in the database, set placeholder as default
	if oldFilePath == "" {
		oldFilePath = "assets/static/images/placeholder.png" // Default placeholder image
	}

	// Чтение файла из формы
	file, header, err := r.FormFile("profile_image")
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer file.Close()

	// Создание пути для сохранения нового изображения
	filePath := fmt.Sprintf("assets/static/images/uploads/%d_%s", userID, header.Filename)

	// Удаление старого файла, если он существует и это не изображение по умолчанию
	if oldFilePath != "assets/static/images/placeholder.png" {
		if err := os.Remove(oldFilePath); err != nil {
			log.Printf("Error deleting the old file: %v", err)
		}
	}

	// Сохранение нового файла на сервере
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error saving file: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer out.Close()

	// Копирование содержимого загруженного файла в созданный файл
	_, err = io.Copy(out, file)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Обновление пути к изображению в базе данных
	_, err = db.Exec("UPDATE users SET profile_image = ? WHERE id = ?", filePath, userID)
	if err != nil {
		log.Printf("Error saving the path to an image in database: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	// Перенаправление на страницу пользователя
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func HandleChangeBio(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		RenderErrorPage(w, r, db, http.StatusMethodNotAllowed, "Method not supported")
		return
	}

	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Error when GetUserIDFromSession HandleChangeBio: %v", err)
		RenderErrorPage(w, r, db, http.StatusUnauthorized, "User is not authorised")
		return
	}

	newBio := r.FormValue("bio")

	_, err = db.Exec("UPDATE users SET bio = ? WHERE id = ?", newBio, userID)
	if err != nil {
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}
