package authentication

import (
	"crypto/subtle"
	"net/http"
)

// SecretAuth implements a simple middleware handler for adding bearer http auth to a route.
func SecretAuth(authSecret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret, ok := secretAuth(r); !ok || !secretMatches(secret, authSecret) {
				secretAuthFailed(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func secretAuthFailed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusUnauthorized)
}

func secretAuth(r *http.Request) (secret string, ok bool) {
	auth := r.Header.Get("Secret")
	if auth == "" {
		return "", false
	}
	return auth, true
}

// secretMatches compares the presented secret with the configured one in
// constant time. An empty configured secret never matches.
func secretMatches(secret, expected string) bool {
	if expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(secret), []byte(expected)) == 1
}
