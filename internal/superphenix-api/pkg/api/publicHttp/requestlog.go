package publicHttp

import (
	stdlog "log"
	"net/http"
	"net/url"
	"os"

	logger "github.com/super-phenix/superphenix/pkg/utils/log"

	"github.com/go-chi/chi/v5/middleware"
)

// redactingLogFormatter wraps chi's default request log formatter and writes
// the request URL without credential query parameters (e.g. ?bearer=).
type redactingLogFormatter struct {
	middleware.LogFormatter
}

func (f redactingLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	redacted := *r
	if u, err := url.ParseRequestURI(r.RequestURI); err == nil {
		redacted.RequestURI = logger.RedactURL(u)
	}
	return f.LogFormatter.NewLogEntry(&redacted)
}

// requestLogger is chi's default request logger with credentials redacted.
var requestLogger = middleware.RequestLogger(redactingLogFormatter{
	LogFormatter: &middleware.DefaultLogFormatter{Logger: stdlog.New(os.Stdout, "", stdlog.LstdFlags), NoColor: false},
})
