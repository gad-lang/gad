package main

import (
	"log"
	"net/http"
	"os"
)

const defaultAddr = ":8080"

func main() {
	app, err := NewApp(".")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	app.routes(mux)
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	log.Printf("cms example listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
