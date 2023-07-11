package util

import "regexp"

func IsAlphaNumeric(s string) bool {
	reg := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	return reg.MatchString(s)
}
