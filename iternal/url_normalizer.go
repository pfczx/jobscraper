package iternal

import (
	"strings"
	"net/url"
)
func urlNormalizer(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.RawQuery = ""
	u.Fragment = ""
	clean := strings.TrimSuffix(u.String(), "/")
	return clean
}
