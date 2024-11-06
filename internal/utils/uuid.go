package utils

import (
	"crypto/rand"
	"fmt"
	"io"
)

func CreateSessionToken() (string, error) {
	// Создаем массив на 16 байтов, что соответствует UUID v4
	uuid := make([]byte, 16)
	// Заполняем массив случайными байтами
	_, err := io.ReadFull(rand.Reader, uuid)
	if err != nil {
		return "", err
	}

	// Настраиваем поля, чтобы соответствовать стандарту UUID v4
	uuid[8] = (uuid[8] & 0xBF) | 0x80 // Устанавливаем два верхних бита на 10
	uuid[6] = (uuid[6] & 0x4F) | 0x40 // Устанавливаем первые 4 бита на 0100

	// Форматируем UUID как строку
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
