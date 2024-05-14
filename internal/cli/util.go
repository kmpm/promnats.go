package cli

import (
	"log/slog"
	"net"
	"net/http"
	"path"
	"strings"
	"time"
)

// shiftPath splits the given path into the first segment (head) and
// the rest (tail). For example, "/foo/bar/baz" gives "foo", "/bar/baz".
func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

func WrapHandler(f http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()

		record := &LogRecord{
			ResponseWriter: w,
		}
		defer func() {
			slog.DebugContext(r.Context(), "response", "method", r.Method, "path", r.URL.Path, "status", record.status, "response_time", time.Since(t))
		}()

		f.ServeHTTP(record, r)

		if record.status >= 400 {
			slog.WarnContext(r.Context(), "bad request", "status", record.status, "method", r.Method, "path", r.URL.Path, "response_time", time.Since(t))
		}

		// if record.status == http.StatusBadRequest {
		// 	log.Warn().Int("status", record.status).Msg("Bad Request")
		// }
	}
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() []string {
	out := []string{}
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				out = append(out, ipnet.IP.String())
			}
		}
	}
	return out
}
