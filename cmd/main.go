package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/sgreben/httpfileserver"
)

const (
	addrEnvVarName           = "ADDR"
	allowUploadsEnvVarName   = "UPLOADS"
	defaultAddr              = ":8080"
	portEnvVarName           = "PORT"
	quietEnvVarName          = "QUIET"
	sslCertificateEnvVarName = "SSL_CERTIFICATE"
	sslKeyEnvVarName         = "SSL_KEY"
)

var Version = ":unknown:"

var addrFlag string
var portFlag int

func addr(cfg httpfileserver.Config) (string, error) {
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

func configureRuntime(cfg httpfileserver.Config) httpfileserver.Config {
	var quietFlag bool

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
	flag.BoolVar(&cfg.AllowUploadsFlag, "uploads", cfg.AllowUploadsFlag, fmt.Sprintf("allow uploads (environment variable %q)", allowUploadsEnvVarName))
	flag.BoolVar(&cfg.AllowUploadsFlag, "u", cfg.AllowUploadsFlag, "(alias for -uploads)")
	flag.Var(&cfg.Routes, "route", cfg.Routes.Help())
	flag.Var(&cfg.Routes, "r", "(alias for -route)")
	flag.StringVar(&cfg.SslCertificate, "ssl-cert", cfg.SslCertificate, fmt.Sprintf("path to SSL server certificate (environment variable %q)", sslCertificateEnvVarName))
	flag.StringVar(&cfg.SslKey, "ssl-key", cfg.SslKey, fmt.Sprintf("path to SSL private key (environment variable %q)", sslKeyEnvVarName))
	flag.Parse()
	if quietFlag {
		log.SetOutput(ioutil.Discard)
	}
	for i := 0; i < flag.NArg(); i++ {
		arg := flag.Arg(i)
		err := cfg.Routes.Set(arg)
		if err != nil {
			log.Fatalf("%q: %v", arg, err)
		}
	}

	return cfg
}

func newConfig() httpfileserver.Config {
	portFlag64, _ := strconv.ParseInt(os.Getenv(portEnvVarName), 10, 64)
	portFlag = int(portFlag64)

	cfg := httpfileserver.NewConfig()
	cfg.AllowUploadsFlag = os.Getenv(allowUploadsEnvVarName) == "true"
	cfg.SslCertificate = os.Getenv(sslCertificateEnvVarName)
	cfg.SslKey = os.Getenv(sslKeyEnvVarName)
	cfg.RootRoute = "/"

	return cfg
}

func main() {
	cfg := configureRuntime(newConfig())
	log.Printf("httpfileserver v%s", Version)

	addr, err := addr(cfg)
	if err != nil {
		log.Fatalf("address/port: %v", err)
	}

	cfg.Addr = addr
	err = httpfileserver.Serve(context.Background(), cfg)
	if err != nil {
		log.Fatalf("start server: %v", err)
	}
}
