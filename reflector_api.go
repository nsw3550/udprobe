package llama

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ReflectorAPI struct {
	server  *http.Server
	handler *http.ServeMux
}

func (api *ReflectorAPI) PromHandler() http.Handler {
	return promhttp.Handler()
}

func (api *ReflectorAPI) StatusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func (api *ReflectorAPI) Run() {
	go func() { log.Fatal(api.server.ListenAndServe()) }()
}

func (api *ReflectorAPI) Stop() {
	api.server.Close()
}

func (api *ReflectorAPI) setupHandlers() {
	api.handler.HandleFunc("/status", api.StatusHandler)
	api.handler.Handle("/metrics", api.PromHandler())
}

func NewReflectorAPI(bind string) *ReflectorAPI {
	handler := http.NewServeMux()
	server := &http.Server{Addr: bind, Handler: handler}
	api := &ReflectorAPI{server: server, handler: handler}
	api.setupHandlers()
	RegisterReflectorPrometheus()
	return api
}
