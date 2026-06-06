package main

import (
	"crypto/tls"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	mw "restapi/internal/api/middlewares"
	"restapi/internal/api/router"
	"restapi/pkg/utils"
	"time"

	"github.com/joho/godotenv"
)

//go:embed .env
var envFile embed.FS

func loadEnvFromEmbeddedFile() {
	content, err := envFile.ReadFile(".env")
	if err != nil {
		log.Fatalf("Error reading .env file: %v", err)
	}

	tempFile, err := os.CreateTemp("", ".env")
	if err != nil {
		log.Fatalf("Error adding a temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(content)
	if err != nil {
		log.Fatalf("Error writing to temp file: %v", err)
	}
	err = tempFile.Close()
	if err != nil {
		log.Fatalf("Error closing temp file: %v", err)
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error closing temp file: %v", err)

	}
}

func main() {
	// err := godotenv.Load()
	// if err != nil {
	// 	return
	// }

	loadEnvFromEmbeddedFile()

	cert := os.Getenv("CERT_FILE")
	key := os.Getenv("KEY_FILE")
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS10,
	}

	rl := mw.NewRateLimiter(2, 5*time.Second)
	hppOptions := mw.HPPOptions{
		CheckBody:               true,
		CheckQuery:              true,
		CheckForOnlyContentType: "x-www-form-urlencoded",
		Whitelist:               []string{"sortOrder", "sortBy", "name", "age", "class"},
	}

	router := router.MainRouter()
	jwtMiddleware := mw.MiddlewaresExcludePaths(mw.JWTMiddleware, "/execs/login", "/execs/forgotpassword", "/execs/resetpassword/reset")
	secureMux := utils.ApplyMiddlewares(router, mw.SecurityHeaders, mw.XSSMiddleware, jwtMiddleware, mw.HPP(hppOptions), mw.Compression, mw.ResponseTimeMiddleware, rl.RateLimiterMiddleware, mw.Cors)

	port := os.Getenv("API_PORT")
	fmt.Println("Server running on port", port)
	server := &http.Server{
		Addr:      port,
		Handler:   secureMux,
		TLSConfig: tlsConfig,
	}

	err := server.ListenAndServeTLS(cert, key)
	if err != nil {
		log.Fatalln("Error running the server:", err)
	}
}
