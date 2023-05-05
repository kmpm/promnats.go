package main

import (
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type LogRecord struct {
	http.ResponseWriter
	status int
}

func (r *LogRecord) Write(p []byte) (int, error) {
	return r.ResponseWriter.Write(p)
}

func (r *LogRecord) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func WrapHandler(f http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()

		record := &LogRecord{
			ResponseWriter: w,
		}
		defer func() {
			log.Debug().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				TimeDiff("response_time", time.Now(), t).
				Msg("response")
		}()

		f.ServeHTTP(record, r)

		if record.status >= 400 {
			log.Warn().
				Int("status", record.status).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				TimeDiff("response_time", time.Now(), t).
				Msg("Bad Request")
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
