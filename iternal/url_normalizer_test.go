package iternal

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
		expected string
	}{
		{
			name:     "urltest1",
			inputURL: "https://www.pracuj.pl/praca/senior-engineer-mobile-android-krakow-kapelanka-42a,oferta,1004500759?s=1f7c2c91&searchId=MTc2NDUyMDk4NTY0MS40NDQ2",
			expected: "https://www.pracuj.pl/praca/senior-engineer-mobile-android-krakow-kapelanka-42a,oferta,1004500759",
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := urlNormalizer(tc.inputURL)
			if actual != tc.expected {
				t.Errorf("Test %v - %s FAIL: expected URL: %v, actual: %v", i, tc.name, tc.expected, actual)
			}
		})
	}
}
