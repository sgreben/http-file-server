package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const (
	addrEnvVarName           = "ADDR"
	allowUploadsEnvVarName   = "UPLOADS"
	defaultAddr              = ":8080"
	portEnvVarName           = "PORT"
	quietEnvVarName          = "QUIET"
	rootRoute                = "/"
	sslCertificateEnvVarName = "SSL_CERTIFICATE"
	sslKeyEnvVarName         = "SSL_KEY"
)

var (
	addrFlag         = os.Getenv(addrEnvVarName)
	allowUploadsFlag = os.Getenv(allowUploadsEnvVarName) == "true"
	portFlag64, _    = strconv.ParseInt(os.Getenv(portEnvVarName), 10, 64)
	portFlag         = int(portFlag64)
	quietFlag        = os.Getenv(quietEnvVarName) == "true"
	routesFlag       routes
	sslCertificate   = os.Getenv(sslCertificateEnvVarName)
	sslKey           = os.Getenv(sslKeyEnvVarName)
)

func init() {
	log.SetFlags(log.LUTC | log.Ldate | log.Ltime)
	log.SetOutput(os.Stderr)
	if addrFlag == "" {
		addrFlag = defaultAddr
	}
	flag.StringVar(&addrFlag, "addr", addrFlag, fmt.Sprintf("address to listen on (environment variable %q)", addrEnvVarName))
	flag.StringVar(&addrFlag, "a", addrFlag, "(alias for -addr)")
	flag.IntVar(&portFlag, "port", portFlag, fmt.Sprintf("port to listen on (overrides -addr port) (environment variable %q)", portEnvVarName))
	flag.IntVar(&portFlag, "p", portFlag, "(alias for -port)")
	flag.BoolVar(&quietFlag, "quiet", quietFlag, fmt.Sprintf("disable all log output (environment variable %q)", quietEnvVarName))
	flag.BoolVar(&quietFlag, "q", quietFlag, "(alias for -quiet)")
	flag.BoolVar(&allowUploadsFlag, "uploads", allowUploadsFlag, fmt.Sprintf("allow uploads (environment variable %q)", allowUploadsEnvVarName))
	flag.BoolVar(&allowUploadsFlag, "u", allowUploadsFlag, "(alias for -uploads)")
	flag.Var(&routesFlag, "route", routesFlag.help())
	flag.Var(&routesFlag, "r", "(alias for -route)")
	flag.StringVar(&sslCertificate, "ssl-cert", sslCertificate, fmt.Sprintf("path to SSL server certificate (environment variable %q)", sslCertificateEnvVarName))
	flag.StringVar(&sslKey, "ssl-key", sslKey, fmt.Sprintf("path to SSL private key (environment variable %q)", sslKeyEnvVarName))
	flag.Parse()
	if quietFlag {
		log.SetOutput(ioutil.Discard)
	}
	for i := 0; i < flag.NArg(); i++ {
		arg := flag.Arg(i)
		err := routesFlag.Set(arg)
		if err != nil {
			log.Fatalf("%q: %v", arg, err)
		}
	}
}

func main() {
	addr, err := addr()
	if err != nil {
		log.Fatalf("address/port: %v", err)
	}
	err = server(addr, routesFlag)
	if err != nil {
		log.Fatalf("start server: %v", err)
	}
}

func server(addr string, routes routes) error {
	mux := http.DefaultServeMux
	handlers := make(map[string]http.Handler)
	paths := make(map[string]string)

	if len(routes.Values) == 0 {
		_ = routes.Set(".")
	}

	for _, route := range routes.Values {
		handlers[route.Route] = &fileHandler{
			route:       route.Route,
			path:        route.Path,
			allowUpload: allowUploadsFlag,
		}
		paths[route.Route] = route.Path
	}

	for route, path := range paths {
		mux.Handle(route, handlers[route])
		log.Printf("serving local path %q on %q", path, route)
	}

	_, rootRouteTaken := handlers[rootRoute]
	if !rootRouteTaken {
		route := routes.Values[0].Route
		mux.Handle(rootRoute, http.RedirectHandler(route, http.StatusTemporaryRedirect))
		log.Printf("redirecting to %q from %q", route, rootRoute)
	}

	binaryPath, _ := os.Executable()
	if binaryPath == "" {
		binaryPath = "server"
	}
	if sslCertificate != "" && sslKey != "" {
		log.Printf("%s (HTTPS) listening on %q", filepath.Base(binaryPath), addr)
		return http.ListenAndServeTLS(addr, sslCertificate, sslKey, mux)
	}
	log.Printf("%s listening on %q", filepath.Base(binaryPath), addr)
	return http.ListenAndServe(addr, mux)
}

func addr() (string, error) {
	portSet := portFlag != 0
	addrSet := addrFlag != ""
	switch {
	case portSet && addrSet:
		a, err := net.ResolveTCPAddr("tcp", addrFlag)
		if err != nil {
			return "", err
		}
		a.Port = portFlag
		return a.String(), nil
	case !portSet && addrSet:
		a, err := net.ResolveTCPAddr("tcp", addrFlag)
		if err != nil {
			return "", err
		}
		return a.String(), nil
	case portSet && !addrSet:
		return fmt.Sprintf(":%d", portFlag), nil
	case !portSet && !addrSet:
		fallthrough
	default:
		return defaultAddr, nil
	}
}
