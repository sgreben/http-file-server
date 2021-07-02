package httpfileserver

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sgreben/httpfileserver/internal/routes"
)

type Config struct {
	Addr             string
	AllowUploadsFlag bool
	RootRoute        string
	SslCertificate   string
	SslKey           string
	Routes           routes.Routes
}

func NewConfig() Config {
	return Config{
		Addr:             ":8080",
		AllowUploadsFlag: false,
		RootRoute:        "/",
		SslCertificate:   "",
		SslKey:           "",
	}
}

func Serve(ctx context.Context, cfg Config) error {
	mux := http.DefaultServeMux
	handlers := make(map[string]http.Handler)
	paths := make(map[string]string)

	if len(cfg.Routes.Values) == 0 {
		_ = cfg.Routes.Set(".")
	}

	for _, route := range cfg.Routes.Values {
		handlers[route.Route] = &fileHandler{
			route:       route.Route,
			path:        route.Path,
			allowUpload: cfg.AllowUploadsFlag,
		}
		paths[route.Route] = route.Path
	}

	for route, path := range paths {
		mux.Handle(route, handlers[route])
		log.Printf("serving local path %q on %q", path, route)
	}

	_, rootRouteTaken := handlers[cfg.RootRoute]
	if !rootRouteTaken {
		route := cfg.Routes.Values[0].Route
		mux.Handle(cfg.RootRoute, http.RedirectHandler(route, http.StatusTemporaryRedirect))
		log.Printf("redirecting to %q from %q", route, cfg.RootRoute)
	}

	binaryPath, _ := os.Executable()
	if binaryPath == "" {
		binaryPath = "server"
	}
	if cfg.SslCertificate != "" && cfg.SslKey != "" {
		log.Printf("%s (HTTPS) listening on %q", filepath.Base(binaryPath), cfg.Addr)
		return http.ListenAndServeTLS(cfg.Addr, cfg.SslCertificate, cfg.SslKey, mux)
	}
	log.Printf("%s listening on %q", filepath.Base(binaryPath), cfg.Addr)
	return http.ListenAndServe(cfg.Addr, mux)
}
