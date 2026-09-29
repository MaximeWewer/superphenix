package log

import "net/url"

// sensitiveQueryParams are URL query parameters that carry credentials.
var sensitiveQueryParams = []string{"bearer", "token", "access_token", "api_key", "apikey"}

// RedactURL returns u as a string with the values of credential-carrying query
// parameters replaced, so it can be logged safely.
func RedactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	if u.RawQuery == "" {
		return u.String()
	}
	query := u.Query()
	changed := false
	for _, key := range sensitiveQueryParams {
		if _, ok := query[key]; ok {
			query.Set(key, "REDACTED")
			changed = true
		}
	}
	if !changed {
		return u.String()
	}
	redacted := *u
	redacted.RawQuery = query.Encode()
	return redacted.String()
}

// StripSensitiveQuery removes credential-carrying query parameters from u, for
// requests forwarded to another service that must not receive them.
func StripSensitiveQuery(u *url.URL) {
	if u == nil || u.RawQuery == "" {
		return
	}
	query := u.Query()
	changed := false
	for _, key := range sensitiveQueryParams {
		if _, ok := query[key]; ok {
			query.Del(key)
			changed = true
		}
	}
	if changed {
		u.RawQuery = query.Encode()
	}
}
