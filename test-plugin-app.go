package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"plugin"
)

var (
	port = flag.Int("p", 8081, "port of the service")
)

type Greeter interface {
	Greet(ctx context.Context) (http.Handler, error)
}

func main() {
	mod := "./plugin/plugin.so"

	// load module
	// 1. open the so file to load the symbols
	plug, err := plugin.Open(mod)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 2. look up a symbol (an exported function or variable)
	// in this case, variable Greeter
	symGreeter, err := plug.Lookup("Greeter")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// 3. Assert that loaded symbol is of a desired type
	// in this case interface type Greeter (defined above)
	var greeter Greeter
	greeter, ok := symGreeter.(Greeter)
	if !ok {
		fmt.Println("unexpected type from module symbol")
		os.Exit(1)
	}

	// 4. use the module

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux, err := greeter.Greet(ctx)
	if err != nil {
		log.Printf("Setting up the gateway: %s", err.Error())
		return
	}

	srvAddr := fmt.Sprintf(":%d", *port)

	s := &http.Server{
		Addr:    srvAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		log.Printf("Shutting down the http server")
		if err := s.Shutdown(context.Background()); err != nil {
			log.Printf("Failed to shutdown http server: %v", err)
		}
	}()

	log.Printf("Starting listening at %s", srvAddr)
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("Failed to listen and serve: %v", err)
	}
}
