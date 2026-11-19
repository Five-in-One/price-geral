package httpserver

import (
	"net/http"
	"os"
)

// instanceName identifies this replica (the container hostname under Docker).
// It is sent back in X-Instance so the gateway logs and the client can show
// which of the load-balanced price-back containers answered.
var instanceName = func() string {
	if v := os.Getenv("INSTANCE_NAME"); v != "" {
		return v
	}
	if h, err := os.Hostname(); err == nil {
		return h
	}
	return "unknown"
}()

func instanceHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Instance", instanceName)
		next.ServeHTTP(w, r)
	})
}
