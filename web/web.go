package web

import (
	"github.com/gorilla/mux"
	"golang.local/master-srv/conf"
	"log"
	"net/http"
	"strconv"
)

func New(yaml conf.ConfigYaml, assignUnit http.Handler, pubkStr string) *http.Server {
	router := mux.NewRouter()
	pkSrv := &PubKeySrv{pubkStr, strconv.Itoa(len(pubkStr))}
	for _, d := range yaml.Listen.Domains {
		router.Host(d).Path(yaml.Listen.GetBasePrefixURL() + "pubkey").Handler(pkSrv)
		router.Host(d).Path(yaml.Listen.GetBasePrefixURL() + "connect").Handler(assignUnit)
		router.Host(d).PathPrefix("/").HandlerFunc(DomainNotAllowed)
	}
	router.PathPrefix("/").HandlerFunc(DomainNotAllowed)
	if yaml.Listen.Web == "" {
		log.Fatalf("[Http] Invalid Listening Address")
	}
	s := &http.Server{
		Addr:         yaml.Listen.Web,
		Handler:      router,
		ReadTimeout:  yaml.Listen.GetReadTimeout(),
		WriteTimeout: yaml.Listen.GetWriteTimeout(),
	}
	go runBackgroundHttp(s)
	return s
}

func runBackgroundHttp(s *http.Server) {
	err := s.ListenAndServe()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("[Http] The http server shutdown successfully")
		} else {
			log.Fatalf("[Http] Error trying to host the http server: %s\n", err.Error())
		}
	}
}

func DomainNotAllowed(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodGet || req.Method == http.MethodHead {
		WriteResponseHeaderCanWriteBody(req.Method, rw, http.StatusOK, "")
	} else {
		rw.Header().Set("Allow", http.MethodOptions+", "+http.MethodGet+", "+http.MethodHead)
		if req.Method == http.MethodOptions {
			WriteResponseHeaderCanWriteBody(req.Method, rw, http.StatusOK, "")
		} else {
			WriteResponseHeaderCanWriteBody(req.Method, rw, http.StatusMethodNotAllowed, "")
		}
	}
}
