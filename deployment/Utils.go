package deployment

import "strings"

func GetTitle(message string) string {
	return strings.Split(message, "\n")[0]
}
