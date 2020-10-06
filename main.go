package main

import (
	"log"
	"net/http"

	"app/app"
)

func main() {
	const addr = "127.0.0.1:8080"

	log.Printf("Initializing...")
	a := app.NewInstance()

	log.Printf("Listening on %s...", addr)
	err := http.ListenAndServe(addr, a)
	if err != nil {
		log.Fatal(err)
	}
}
