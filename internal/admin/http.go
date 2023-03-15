package admin

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/yohamta/dagu/internal/admin/handlers"
	"github.com/yohamta/dagu/internal/config"
	"github.com/yohamta/dagu/internal/utils"
)

type server struct {
	config          *config.Config
	addr            string
	server          *http.Server
	idleConnsClosed chan struct{}
}

func NewServer(cfg *config.Config) *server {
	return &server{
		addr:            net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		config:          cfg,
		idleConnsClosed: nil,
	}
}

func (svr *server) Shutdown() {
	err := svr.server.Shutdown(context.Background())
	if err != nil {
		log.Printf("server shutdown: %v", err)
	}
	if svr.idleConnsClosed != nil {
		close(svr.idleConnsClosed)
		svr.idleConnsClosed = nil
	}
}

func (svr *server) Serve() (err error) {
	svr.setupServer()
	svr.setupHandler()

	svr.idleConnsClosed = make(chan struct{})

	host := utils.StringWithFallback(svr.config.Host, "localhost")
	log.Printf("admin server is running at \"http://%s:%d\"\n",
		host, svr.config.Port)

	err = svr.server.ListenAndServe()
	if err == http.ErrServerClosed {
		err = nil
	}
	if err != nil {
		return err
	}

	<-svr.idleConnsClosed

	log.Printf("server closed")

	return
}

func (svr *server) setupServer() {
	svr.server = &http.Server{
		Addr: svr.addr,
	}
}

func (svr *server) setupHandler() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Access-Control-Allow-Origin", "*")
				w.Header().Add("Access-Control-Allow-Methods", "*")
				w.Header().Add("Access-Control-Allow-Headers", "*")
				h.ServeHTTP(w, r)
			})
	})

	if svr.config.IsBasicAuth {
		r.Use(middleware.BasicAuth(
			"restricted",
			map[string]string{
				svr.config.BasicAuthUsername: svr.config.BasicAuthPassword,
			},
		))
	}

	handlers.ConfigRoutes(r)

	r.Post("/shutdown", svr.handleShutdown)

	svr.server.Handler = r
}

func (svr *server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	log.Println("received shutdown request")
	_, _ = w.Write([]byte("shutting down the dagu server...\n"))
	go func() {
		svr.Shutdown()
	}()
}
