package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
)

//func HandleIndex(w http.ResponseWriter, r *http.Request, db *sql.DB) {

func HandleErrorPage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	tmpl, err := template.ParseFiles("assets/template/error.html")
	if err != nil {
		log.Printf("Ошибка загрузки шаблона ошибки: %v", err)
		http.Error(w, "Ошибка загрузки страницы ошибки", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, nil)
}

// // ErrorHandler renders error messages for HTTP 4XX and 5XX errors
// func ErrorHandler(w http.ResponseWriter, r *http.Request, status int) {
// 	//func ErrorHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, status int) {
// 	w.WriteHeader(status)

// 	var errorMessage string
// 	var errorTitle string
// 	var imagePath string

// 	switch status {
// 	case http.StatusBadRequest:
// 		errorTitle = "400 - Bad Request"
// 		errorMessage = "The server couldn't process your request. Please check your input."
// 		imagePath = "/assets/images/4xx-error.png"
// 	case http.StatusUnauthorized:
// 		errorTitle = "401 - Unauthorized"
// 		errorMessage = "You need to log in to access this resource."
// 		imagePath = "/assets/images/4xx-error.png"
// 	case http.StatusForbidden:
// 		errorTitle = "403 - Forbidden"
// 		errorMessage = "You do not have permission to access this resource."
// 		imagePath = "/assets/images/4xx-error.png"
// 	case http.StatusNotFound:
// 		errorTitle = "404 - Not Found"
// 		errorMessage = "The page you're looking for couldn't be found."
// 		imagePath = "/assets/images/4xx-error.png"
// 	case http.StatusMethodNotAllowed:
// 		errorTitle = "405 - Method Not Allowed"
// 		errorMessage = "The method you used is not allowed for this endpoint."
// 		imagePath = "/assets/images/4xx-error.png"
// 	case http.StatusInternalServerError:
// 		errorTitle = "500 - Internal Server Error"
// 		errorMessage = "Something went wrong on the server. We're working to fix it!"
// 		imagePath = "/assets/images/5xx-error.png"
// 	case http.StatusServiceUnavailable:
// 		errorTitle = "503 - Service Unavailable"
// 		errorMessage = "The server is currently unavailable. Please try again later."
// 		imagePath = "/assets/images/5xx-error.png"
// 	default:
// 		errorTitle = "Error"
// 		errorMessage = "An unexpected error occurred. Please try again later."
// 		imagePath = "/assets/images/default-error.png"
// 	}

// 	// Parse the error template
// 	tmpl, err := template.ParseFiles("./assets/template/error.html")
// 	if err != nil {
// 		log.Println("Error parsing error template:", err)
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	}

// 	// Проверка на наличие сессии пользователя
// 	// var user *models.User
// 	// cookie, err := r.Cookie("session_token")
// 	// if err == nil {
// 	// 	var userID int
// 	// 	err = db.QueryRow("SELECT user_id FROM sessions WHERE session_token = ?", cookie.Value).Scan(&userID)
// 	// 	if err == nil {
// 	// 		user = &models.User{}
// 	// 		err = db.QueryRow("SELECT id, username FROM users WHERE id = ?", userID).Scan(&user.ID, &user.Username)
// 	// 		if err != nil {
// 	// 			log.Printf("Ошибка при получении пользователя: %v", err)
// 	// 		}
// 	// 	}
// 	// }

// 	// // Создаем структуру для передачи в шаблон
// 	// pageData := models.IndexPageData{
// 	// 	Posts: posts,
// 	// 	User:  user, // может быть nil, если пользователь не залогинен
// 	// }

// 	// tmpl, err := template.ParseFiles("assets/template/header.html", "assets/template/index.html")
// 	// if err != nil {
// 	// 	log.Printf("Ошибка загрузки шаблона: %v", err)
// 	// 	http.Error(w, "Ошибка загрузки шаблона", http.StatusInternalServerError)
// 	// 	return
// 	// }

// 	// // Set the content type
// 	// w.Header().Set("Content-Type", "text/html")

// 	// // Execute the "index" template as the main entry point
// 	// err = tmpl.ExecuteTemplate(w, "index", pageData) // specify "index" here
// 	// if err != nil {
// 	// 	log.Printf("Ошибка рендеринга: %v", err)
// 	// 	http.Error(w, "Ошибка рендеринга страницы", http.StatusInternalServerError)
// 	// 	return
// 	// }

// 	// Render the error page
// 	err = tmpl.Execute(w, struct {
// 		StatusCode   int
// 		ErrorTitle   string
// 		ErrorMessage string
// 		ImagePath    string
// 	}{
// 		StatusCode:   status,
// 		ErrorTitle:   errorTitle,
// 		ErrorMessage: errorMessage,
// 		ImagePath:    imagePath,
// 	})

// 	if err != nil {
// 		log.Println("Error rendering template:", err)
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 	}
// }

// // REDO!!!!!!!!!!!!!!!!!!!!!
// // Пример использования методов HTTP с обработкой ошибок
// func ExampleHandler(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	case "GET":
// 		// Логика обработки GET запроса
// 		http.ServeFile(w, r, "index.html")
// 	case "POST":
// 		// Логика обработки POST запроса
// 		err := r.ParseForm()
// 		if err != nil {
// 			ErrorHandler(w, r, http.StatusBadRequest) // Ошибка 400
// 			return
// 		}
// 		// Обработка данных формы
// 		w.Write([]byte("Form submitted successfully"))
// 	case "PUT":
// 		// Логика обработки PUT запроса
// 		// Проверка прав доступа или корректности данных
// 		ErrorHandler(w, r, http.StatusForbidden) // Ошибка 403, если нет прав
// 	case "DELETE":
// 		// Логика обработки DELETE запроса
// 		// Удаление ресурса
// 		ErrorHandler(w, r, http.StatusNotFound) // Ошибка 404, если ресурс не найден
// 	default:
// 		ErrorHandler(w, r, http.StatusMethodNotAllowed) // Ошибка 405
// 	}
// }
