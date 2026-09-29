package helper

import (
	"net/http"
	"strings"
)

// RejectAmbiguousPath refuses requests whose path contains encoded separators
// (%2F, %5C), backslashes or dot segments. Such paths route differently once
// decoded or cleaned, so a path parameter could otherwise escape its segment
// and reach another organization or project when the request is proxied.
func RejectAmbiguousPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsAmbiguousPath(r) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// IsAmbiguousPath reports whether the request path contains an encoded
// separator, a backslash or a "." / ".." segment.
func IsAmbiguousPath(r *http.Request) bool {
	escaped := strings.ToLower(r.URL.EscapedPath())
	if strings.Contains(escaped, "%2f") || strings.Contains(escaped, "%5c") {
		return true
	}
	if strings.Contains(r.URL.Path, `\`) {
		return true
	}
	for _, segment := range strings.Split(r.URL.Path, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}
