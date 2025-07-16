package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// options contains certain options which a http server needs while
// building a new server
type options struct {
	port      string
	handler   http.Handler
	errorlog  *log.Logger
	tlsconfig *tls.Config
}

// Option is a function that used in NewServer function,
// which will execute each option and record results into
// a variable
type Option func(options *options) error

// NewServer creates a new server with a certain address and extra options
func NewServer(addr string, opts ...Option) (*http.Server, error) {
	var options options
	for _, opt := range opts {
		if err := opt(&options); err != nil {
			return nil, err
		}
	}

	var port string
	if options.port == "" { //If port has not been defined,
		port = "4000" //then use default value 4000
	} else {
		if options.port == "0" {
			port = "4000"
		} else {
			port = options.port
		}
	}
	addr += ":" + port

	if options.handler == nil {
		options.handler = http.NewServeMux()
	}
	if options.errorlog == nil {
		prefix := fmt.Sprintf("Error at server %s\t", addr)
		options.errorlog = log.New(os.Stderr, prefix, log.Ldate|log.Ltime|log.Llongfile)
	}
	if options.tlsconfig == nil {
	} // Do Nothig

	return &http.Server{
		Addr:                         addr,
		Handler:                      options.handler,
		DisableGeneralOptionsHandler: false,
		TLSConfig:                    options.tlsconfig,
		ReadTimeout:                  5 * time.Second,
		ReadHeaderTimeout:            0,
		WriteTimeout:                 10 * time.Second,
		IdleTimeout:                  time.Minute,
		MaxHeaderBytes:               0,
		TLSNextProto:                 nil,
		ConnState:                    nil,
		ErrorLog:                     options.errorlog,
		BaseContext:                  nil,
		ConnContext:                  nil,
		HTTP2:                        nil,
		Protocols:                    nil,
	}, nil
}

func WithPort(port string) Option {
	return func(options *options) error {
		if p, err := strconv.Atoi(port); err != nil {
			return err
		} else {
			if p < 0 {
				return errors.New("port should be greater than 0")
			}
			options.port = port
			return nil
		}
	}
}

func WithHandler(h http.Handler) Option {
	return func(options *options) error {
		if h == nil {
			return errors.New("handler cannot be nil")
		}
		options.handler = h
		return nil
	}
}

func WithErrorLog(log *log.Logger) Option {
	return func(options *options) error {
		if log == nil {
			return errors.New("logger cannot be nil")
		}
		options.errorlog = log
		return nil
	}
}

func WithTLSConfig(config *tls.Config) Option {
	return func(options *options) error {
		if config == nil {
			return errors.New("config cannot be nil")
		}
		options.tlsconfig = config
		return nil
	}
}
