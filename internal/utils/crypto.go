package utils

//package models

// import (
// 	"crypto/rand"
// 	"crypto/sha256"
// 	"database/sql"
// 	"encoding/base64"
// 	"fmt"

// 	database "literary-lions/internal/db"
// )

// var db *sql.DB

// func Init() error {
// 	db = database.InitDB("forum.db") // Исправлено для одного возвращаемого значения
// 	if db == nil {
// 		return fmt.Errorf("failed to initialize database")
// 	}
// 	return nil
// }

// // Закрытие соединения с базой данных
// func Close() {
// 	if db != nil {
// 		db.Close()
// 	}
// }

// // Генерация случайной соли
// func generateSalt() (string, error) {
// 	salt := make([]byte, 16) // Создаем 16 байтов для соли
// 	_, err := rand.Read(salt)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to generate salt: %v", err)
// 	}
// 	return base64.StdEncoding.EncodeToString(salt), nil
// }

// // Хэширование пароля с использованием соли
// func hashPassword(password, salt string) string {
// 	hasher := sha256.New()
// 	hasher.Write([]byte(password + salt)) // Хэшируем пароль с добавлением соли
// 	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
// }

// // Создание пользователя с безопасным хэшированием пароля
// func CreateUser(email, username, password string) error {
// 	// Генерируем соль
// 	salt, err := generateSalt()
// 	if err != nil {
// 		return err
// 	}

// 	// Хэшируем пароль с солью
// 	hashedPassword := hashPassword(password, salt)

// 	// Используем транзакцию для обеспечения целостности данных
// 	tx, err := db.Begin()
// 	if err != nil {
// 		return fmt.Errorf("failed to begin transaction: %v", err)
// 	}

// 	_, err = tx.Exec("INSERT INTO users (email, username, password, salt) VALUES (?, ?, ?, ?)", email, username, hashedPassword, salt)
// 	if err != nil {
// 		tx.Rollback() // Откатываем транзакцию в случае ошибки
// 		return fmt.Errorf("failed to insert user: %v", err)
// 	}

// 	// Завершаем транзакцию
// 	err = tx.Commit()
// 	if err != nil {
// 		return fmt.Errorf("failed to commit transaction: %v", err)
// 	}

// 	return nil
// }

// // Проверка пароля
// func CheckPassword(email, password string) (bool, error) {
// 	var hashedPassword, salt string
// 	err := db.QueryRow("SELECT password, salt FROM users WHERE email = ?", email).Scan(&hashedPassword, &salt)
// 	if err != nil {
// 		return false, fmt.Errorf("failed to retrieve user data: %v", err)
// 	}

// 	// Хэшируем введенный пароль с извлеченной солью
// 	expectedPassword := hashPassword(password, salt)
// 	return hashedPassword == expectedPassword, nil
// }
