package utils

import "strings"

func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func SanitizeFileName(fileName string) string {
	invalidChars := []struct {
		old, new string
	}{
		{"<", "_"}, {">", "_"}, {":", "_"}, {"\"", "_"}, {"/", "_"},
		{"\\", "_"}, {"|", "_"}, {"?", "_"}, {"*", "_"},
	}

	for _, char := range invalidChars {
		fileName = strings.ReplaceAll(fileName, char.old, char.new)
	}
	return fileName
}
