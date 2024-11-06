package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

// Структура для хранения капчи
type Captcha struct {
	Question  string
	Answer    string
	ExpiresAt time.Time
}

// Генерация случайного математического выражения
func GenerateCaptcha() Captcha {
	rand.Seed(time.Now().UnixNano())
	var a, b, result int
	var op string

	for {
		a = rand.Intn(10)
		b = rand.Intn(10)
		switch rand.Intn(4) {
		case 0:
			op = "+"
			result = a + b
		case 1:
			op = "-"
			result = a - b
		case 2:
			op = "*"
			result = a * b
		case 3:
			if b != 0 && a%b == 0 {
				op = "/"
				result = a / b
			} else {
				continue
			}
		}
		if result >= 0 && result <= 9 {
			break
		}
	}

	question := fmt.Sprintf("%d %s %d = ?", a, op, b)
	hashedAnswer := hashAnswer(fmt.Sprintf("%d", result))

	// Капча с хешированным ответом и временем жизни (например, 1 минута)
	return Captcha{
		Question:  question,
		Answer:    hashedAnswer,
		ExpiresAt: time.Now().Add(1 * time.Minute),
	}
}

// Хеширование ответа
func hashAnswer(answer string) string {
	hash := sha256.Sum256([]byte(answer))
	return hex.EncodeToString(hash[:])
}

// Проверка капчи
// func VerifyCaptcha(input string, captcha Captcha) bool {
// 	// Проверяем срок действия капчи
// 	if time.Now().After(captcha.ExpiresAt) {
// 		fmt.Println("Captcha expired")
// 		return false
// 	}

// 	// Проверяем хеш ответа
// 	return hashAnswer(input) == captcha.Answer
// }

// Проверка капчи
func VerifyCaptcha(input string, captcha Captcha) bool {
	// Проверяем срок действия капчи
	if time.Now().After(captcha.ExpiresAt) {
		fmt.Println("Captcha expired")
		return false
	}

	// Проверяем хеш ответа
	return hashAnswer(input) == captcha.Answer
}

// func main() {
// 	// Генерация капчи
// 	captcha := GenerateCaptcha()
// 	fmt.Println("Captcha Question:", captcha.Question)

// 	// Пользовательский ввод
// 	var input string
// 	fmt.Print("Введите ответ: ")
// 	fmt.Scanln(&input)

// 	// Проверка капчи
// 	if VerifyCaptcha(input, captcha) {
// 		fmt.Println("Капча пройдена!")
// 	} else {
// 		fmt.Println("Неверный ответ или срок капчи истек.")
// 	}
// }
