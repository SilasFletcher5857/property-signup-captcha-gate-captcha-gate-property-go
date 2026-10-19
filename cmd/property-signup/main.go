package main

import (
	"log"
	"net/http"
	"os"

	"github.com/example/property-signup-captcha/internal/property"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	service := property.NewSignupService(property.NewCaptchaClient(apiKey))
	mux := http.NewServeMux()
	mux.Handle("POST /signup", property.SignupHandler{Service: service})

	address := ":8080"
	log.Printf("property signup service listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}
