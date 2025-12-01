package util

import (
	"regexp"
	"strings"
)

func PascalCaseToSnakeCase(s string) string {
	re := regexp.MustCompile("([A-Z][a-z0-9]*)")
	snake := re.ReplaceAllStringFunc(s, func(sub string) string {
		return "_" + strings.ToLower(sub)
	})
	snake = strings.TrimLeft(snake, "_")
	return snake
}
