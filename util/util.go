package util

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

func IsValidWebhook(url string) bool {
	return false
}

func IsErrNoRecords(err error) bool {
	return strings.Contains(err.Error(), "sql: no rows in result set")
}

func IsValidDomainName(domain string) bool {
	re := regexp.MustCompile(`^([a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]\.)+[a-zA-Z]{2,6}$`)
	return re.MatchString(domain)
}

func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(email)
}

// Thank you ChatGPT
func IsValidPassword(password string) bool {
	if len(password) < 7 {
		return false
	}
	var (
		hasUpper   = false
		hasLower   = false
		hasDigit   = false
		hasSpecial = false
	)
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

func Pluralize(word string, amount int) string {
	if amount == 1 {
		return word
	}

	return word + "s"
}

func DaysLeft(t time.Time) string {
	timeZero := time.Time{}
	if t.Equal(timeZero) {
		return "n/a"
	}
	return fmt.Sprintf("%d days", time.Until(t)/(time.Hour*24))
}
