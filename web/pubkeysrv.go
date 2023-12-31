package web

import (
	"net/http"
)

type PubKeySrv struct {
	Key    string
	KeyLen string
}

func (p *PubKeySrv) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("Cache-Control", "max-age=0, no-cache, no-store, must-revalidate")
	writer.Header().Set("Pragma", "no-cache")
	if request.Method == http.MethodGet {
		writer.Header().Set("Content-Type", "application/x-pem-file")
		writer.Header().Set("Content-Length", p.KeyLen)
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(p.Key))
	} else if request.Method == http.MethodOptions {
		writer.Header().Set("Allow", http.MethodOptions+", "+http.MethodGet)
		writer.WriteHeader(http.StatusOK)
	} else {
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}
