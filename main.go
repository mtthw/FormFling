package main

import (
	"log"
	"net/http"

	"formfling/internal/config"
	"formfling/internal/handlers"
	"formfling/internal/middleware"
	"formfling/internal/services"

	"github.com/gorilla/mux"
)

func main() {
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

	// Initialize services
	emailService := services.NewEmailService(cfg)

	// Setup handlers
	submitHandler := handlers.NewSubmitHandler(emailService, rateLimit, trustedProxies...)
	healthHandler := handlers.NewHealthHandler()
	statusHandler := handlers.NewStatusHandler(cfg)

	// Setup router
	r := mux.NewRouter()
	r.Use(middleware.CORS(cfg))

	r.HandleFunc("/submit", submitHandler.Handle).Methods("POST", "OPTIONS")
	r.HandleFunc("/health", healthHandler.Handle).Methods("GET")
	r.HandleFunc("/status", statusHandler.Handle).Methods("GET")

	// Static file serving
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/static/")))

	log.Printf("FormFling server starting on port %s", cfg.Port)
	log.Printf("Allowed origins: %v", cfg.AllowedOrigins)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: config.HTTPReadHeaderTimeout,
		ReadTimeout:       config.HTTPReadTimeout,
		WriteTimeout:      config.HTTPWriteTimeout,
		IdleTimeout:       config.HTTPIdleTimeout,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
