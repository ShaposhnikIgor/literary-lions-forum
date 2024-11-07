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
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// userID, err := GetUserIDFromSession(r, db)
	// if err != nil {
	// 	log.Printf("Ошибка при GetUserIDFromSession user page: %v", err)
	// 	http.Error(w, "Ошибка сессии", http.StatusUnauthorized)
	// 	return
	// }

	// var user models.User
	// err = db.QueryRow("SELECT username, email, COALESCE(bio, ''), COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&user.Username, &user.Email, &user.Bio, &user.ProfImage)
	// fmt.Println(user)
	// if err != nil {
	// 	log.Printf("Пользователь не найден: %v", err)
	// 	http.Error(w, "Пользователь не найден", http.StatusInternalServerError)
	// 	return
	// }

	// tmpl, err := template.ParseFiles("assets/template/user.html")
	// if err != nil {
	// 	http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
	// 	return
	// }

	// tmpl.Execute(w, user)

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

	//Создаем структуру для передачи в шаблон
	pageData := models.UserPageData{
		User:       user, // может быть nil, если пользователь не залогинен
		Categories: categories,
	}

	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/user.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "text/html")

	// Execute the "index" template as the main entry point
	err = tmpl.ExecuteTemplate(w, "user", pageData)
	if err != nil {
		log.Printf("Ошибка рендеринга: %v", err)
		http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
		return
	}
}

// func HandleUserComments(w http.ResponseWriter, r *http.Request, db *sql.DB) {
// 	if r.Method != http.MethodGet {
// 		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	userID, err := GetUserIDFromSession(r, db)
// 	if err != nil {
// 		log.Printf("Ошибка при GetUserIDFromSession HandleUserComments: %v", err)
// 		http.Error(w, "Ошибка сессии", http.StatusUnauthorized)
// 		return
// 	}

// 	// Получаем комментарии пользователя
// 	rows, err := db.Query("SELECT body, created_at FROM comments WHERE user_id = ?", userID)
// 	if err != nil {
// 		http.Error(w, "Ошибка при извлечении комментариев", http.StatusInternalServerError)
// 		return
// 	}
// 	defer rows.Close()

// 	var comments []models.Comment
// 	for rows.Next() {
// 		var comment models.Comment
// 		if err := rows.Scan(&comment.Body, &comment.CreatedAt); err != nil {
// 			http.Error(w, "Ошибка при чтении комментариев", http.StatusInternalServerError)
// 			return
// 		}
// 		comments = append(comments, comment)
// 	}

// 	tmpl, err := template.ParseFiles("assets/template/user_comments.html") //надо сделать страницу html
// 	if err != nil {
// 		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
// 		return
// 	}

// 	tmpl.Execute(w, comments)
// }

// func HandleUserLikes(w http.ResponseWriter, r *http.Request, db *sql.DB) {
// 	if r.Method != http.MethodGet {
// 		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	userID, err := GetUserIDFromSession(r, db)
// 	if err != nil {
// 		log.Printf("Ошибка при GetUserIDFromSession HandleUserLikes: %v", err)
// 		http.Error(w, "Ошибка сессии", http.StatusUnauthorized)
// 		return
// 	}

// 	rows, err := db.Query("SELECT target_id, target_type, is_like FROM likes_dislikes WHERE user_id = ?", userID)
// 	if err != nil {
// 		http.Error(w, "Ошибка при извлечении лайков", http.StatusInternalServerError)
// 		return
// 	}
// 	defer rows.Close()

// 	var likes []models.LikeDislike
// 	for rows.Next() {
// 		var like models.LikeDislike
// 		if err := rows.Scan(&like.TargetID, &like.TargetType, &like.IsLike); err != nil {
// 			http.Error(w, "Ошибка при чтении лайков", http.StatusInternalServerError)
// 			return
// 		}
// 		likes = append(likes, like)
// 	}

// 	tmpl, err := template.ParseFiles("assets/template/user_likes.html") //надо сделать страницу html
// 	if err != nil {
// 		http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
// 		return
// 	}

// 	tmpl.Execute(w, likes)
// }

func HandleChangeUsername(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Ошибка при GetUserIDFromSession HandleChangeUsername: %v", err)
		http.Error(w, "Ошибка сессии", http.StatusUnauthorized)
		return
	}

	newUsername := r.FormValue("username")

	_, err = db.Exec("UPDATE users SET username = ? WHERE id = ?", newUsername, userID)
	if err != nil {
		http.Error(w, "Ошибка при обновлении имени пользователя", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func HandleChangePassword(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получение ID пользователя из сессии
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Ошибка при GetUserIDFromSession в HandleChangePassword: %v", err)
		http.Error(w, "Ошибка сессии", http.StatusUnauthorized)
		return
	}

	// Чтение и проверка текущего пароля и нового пароля из формы
	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if newPassword != confirmPassword {
		http.Error(w, "Пароли не совпадают", http.StatusBadRequest)
		return
	}

	// Извлечение хеша текущего пароля из базы данных для проверки
	var passwordHash string
	err = db.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		log.Printf("Ошибка при получении password_hash: %v", err)
		http.Error(w, "Ошибка при обработке запроса", http.StatusInternalServerError)
		return
	}

	// Проверка текущего пароля
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword))
	if err != nil {
		http.Error(w, "Неверный текущий пароль", http.StatusBadRequest)
		return
	}

	// Хеширование нового пароля
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Ошибка при хешировании пароля", http.StatusInternalServerError)
		return
	}

	// Обновление пароля в базе данных
	_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHashedPassword, userID)
	if err != nil {
		log.Printf("Ошибка при обновлении пароля: %v", err)
		http.Error(w, "Ошибка при обновлении пароля", http.StatusInternalServerError)
		return
	}

	// Перенаправление на страницу профиля после успешного обновления пароля
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

// func HandleUploadProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
// 	if r.Method != http.MethodPost {
// 		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	file, _, err := r.FormFile("profile_image")
// 	if err != nil {
// 		http.Error(w, "Ошибка загрузки файла", http.StatusInternalServerError)
// 		return
// 	}
// 	defer file.Close()

// 	// Здесь можно сохранить файл на сервере
// 	// Пример: ioutil.WriteFile("/path/to/images/" + handler.Filename, file, 0644)

// 	http.Redirect(w, r, "/user", http.StatusSeeOther)
// }

func HandleUploadProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получение ID пользователя из сессии
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Ошибка получения ID пользователя из сессии: %v", err)
		http.Error(w, "Ошибка сессии", http.StatusUnauthorized)
		return
	}

	// Чтение файла из формы
	file, header, err := r.FormFile("profile_image")
	if err != nil {
		http.Error(w, "Ошибка загрузки файла", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Создание пути для сохранения изображения
	filePath := fmt.Sprintf("uploads/%d_%s", userID, header.Filename)

	// Сохранение файла на сервере
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error saving file: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	// Копирование содержимого загруженного файла в созданный файл
	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Ошибка копирования файла", http.StatusInternalServerError)
		return
	}

	// Обновление пути к изображению в базе данных
	_, err = db.Exec("UPDATE users SET profile_image = ? WHERE id = ?", filePath, userID)
	if err != nil {
		log.Printf("Ошибка сохранения пути к изображению в базе данных: %v", err)
		http.Error(w, "Ошибка сохранения изображения", http.StatusInternalServerError)
		return
	}

	// Перенаправление на страницу пользователя
	http.Redirect(w, r, "/user", http.StatusSeeOther)
}

func ServeProfileImage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		http.Error(w, "Ошибка сессии", http.StatusUnauthorized)
		return
	}

	// Извлечение пути к изображению из базы данных
	var filePath string
	err = db.QueryRow("SELECT COALESCE(profile_image, '') FROM users WHERE id = ?", userID).Scan(&filePath)
	if err != nil {
		log.Printf("Ошибка при извлечении пути к изображению: %v", err)
		http.Error(w, "Изображение не найдено", http.StatusNotFound)
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
		http.Error(w, "Неподдерживаемый формат изображения", http.StatusUnsupportedMediaType)
		return
	}

	// Отправка файла
	http.ServeFile(w, r, filePath)
}

func HandleChangeBio(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	userID, err := GetUserIDFromSession(r, db)
	if err != nil {
		log.Printf("Ошибка при GetUserIDFromSession HandleChangeBio: %v", err)
		http.Error(w, "Ошибка сессии", http.StatusUnauthorized)
		return
	}

	newBio := r.FormValue("bio")

	_, err = db.Exec("UPDATE users SET bio = ? WHERE id = ?", newBio, userID)
	if err != nil {
		http.Error(w, "Ошибка при обновлении имени пользователя", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/user", http.StatusSeeOther)
}
