package handlers

import (
	"database/sql"
	"html/template"
	models "literary-lions/internal/models"
	"log"
	"net/http"
)

func RenderErrorPage(w http.ResponseWriter, r *http.Request, db *sql.DB, status int, message string) {
	HandleErrorPage(w, r, db, status, message)
}

func HandleErrorPage(w http.ResponseWriter, r *http.Request, db *sql.DB, status int, message string) {
	// Установите статус ответа
	w.WriteHeader(status)

	// Получите данные пользователя и категорий для заголовка
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

	// Получите категории
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

	// Подготовьте данные для шаблона
	pageData := models.ErrorPageData{
		ErrorTitle:   http.StatusText(status),
		ErrorMessage: message,
		User:         user,
		Categories:   categories,
	}

	// Парсинг и рендеринг шаблона
	tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/error.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка загрузки шаблона")
		return
	}
	if err := tmpl.ExecuteTemplate(w, "error", pageData); err != nil {
		log.Printf("Ошибка рендеринга страницы ошибки: %v", err)
		RenderErrorPage(w, r, db, http.StatusInternalServerError, "Ошибка рендеринга страницы ошибки")
		return
	}
}
