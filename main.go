package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"formfling/internal/config"
	"formfling/internal/handlers"
	"formfling/internal/middleware"
	"formfling/internal/services"
)

// Set at build time from the nearest Git release tag.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("FormFling %s\n", version)
		return
	}

	// Load configuration
	cfg := config.Load()

	// Validate required environment variables
	if cfg.SMTPUsername == "" || cfg.SMTPPassword == "" {
		log.Fatal("SMTP_USERNAME and SMTP_PASSWORD are required")
	}
	if cfg.FromEmail == "" || cfg.ToEmail == "" {
		log.Fatal("FROM_EMAIL and TO_EMAIL are required")
	}
	if err := config.ValidateStatusRedirectURL(cfg.StatusRedirectURL); err != nil {
		log.Fatal(err)
	}
	trustedProxies, err := config.ParseTrustedProxyCIDRs(cfg.TrustedProxyCIDRs)
	if err != nil {
		log.Fatal(err)
	}
	rateLimit, err := config.ParseRateLimitSettings(cfg.RateLimitMaxEmails, cfg.RateLimitWindow)
	if err != nil {
		log.Fatal(err)
	}
	exposeVersion, err := config.ParseExposeVersion(cfg.ExposeVersion)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize services
	emailService := services.NewEmailService(cfg)

	// Setup handlers
	submitHandler := handlers.NewSubmitHandler(emailService, rateLimit, cfg.StatusRedirectURL, trustedProxies...)
	publicVersion := ""
	if exposeVersion {
		publicVersion = version
	}
	healthHandler := handlers.NewHealthHandler(publicVersion)

	// Setup router
	r := http.NewServeMux()
	r.HandleFunc("POST /submit", submitHandler.Handle)
	r.HandleFunc("GET /health", healthHandler.Handle)

	// Static file serving
	r.Handle("/", http.FileServer(http.Dir("./web/static/")))

	log.Printf("FormFling %s starting on port %s", version, cfg.Port)
	log.Printf("Allowed origins: %v", cfg.AllowedOrigins)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           middleware.CORS(cfg)(r),
		ReadHeaderTimeout: config.HTTPReadHeaderTimeout,
		ReadTimeout:       config.HTTPReadTimeout,
		WriteTimeout:      config.HTTPWriteTimeout,
		IdleTimeout:       config.HTTPIdleTimeout,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
