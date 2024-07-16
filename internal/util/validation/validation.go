package validation

import (
	"errors"
	"fmt"
	"github.com/pariz/gountries"
	"regexp"
)

func IsEmailValid(email string) bool {
	var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func ValidateCountryCode(countryCode string) error {
	filter := gountries.New()
	_, err := filter.FindCountryByAlpha(countryCode)
	if err != nil {
		return errors.New(fmt.Sprintf("invalid country code: %s", countryCode))
	}
	return nil
}
