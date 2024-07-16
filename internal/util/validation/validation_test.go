package validation

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCountryCodeValidation(t *testing.T) {

	valid := []string{"DK", "GB", "UKR", "GY", "VA"}
	for _, s := range valid {
		require.NoError(t, ValidateCountryCode(s))
	}

	// the country code GB is used for the United Kingdom in the ISO 3166-1 alpha-2 standard
	invalid := []string{"UN", "UK", "LW", "AA"}
	for _, s := range invalid {
		require.Error(t, ValidateCountryCode(s))
	}
}

func TestEmailValidation(t *testing.T) {

	valid := []string{"kk@kk.dk", "john@email.co.uk", "foo@bar.net"}
	for _, s := range valid {
		require.True(t, IsEmailValid(s))
	}

	invalid := []string{"no@domain", "noatemail.com", "nothing", "double@@at.com", "invalid@domain.n"}
	for _, s := range invalid {
		require.False(t, IsEmailValid(s))
	}

}
