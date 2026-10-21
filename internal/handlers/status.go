package handlers

import (
	"html/template"
	"log"
	"net/http"

	"formfling/internal/config"
)

type StatusHandler struct {
	config         *config.Config
	statusTemplate *template.Template
}

func NewStatusHandler(cfg *config.Config) *StatusHandler {
	statusTemplate, err := template.ParseFiles(cfg.StatusTemplate)
	if err != nil {
		log.Fatal("Error loading status template:", err)
	}

	return &StatusHandler{
		config:         cfg,
		statusTemplate: statusTemplate,
	}
}

type StatusPageData struct {
	Status      string
	FormTitle   string
	Message     string
	RedirectURL string
}

func (h *StatusHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	status := r.URL.Query().Get("type")
	if status == "" {
		status = "success" // default
	}

	var message string
	if status == "success" {
		message = "Your message has been sent successfully!"
	} else if status == "rate_limited" {
		message = "You have reached the message limit. Please try again later."
	} else {
		message = "There was an error sending your message. Please try again."
	}

	data := StatusPageData{
		Status:      status,
		FormTitle:   h.config.FormTitle,
		Message:     message,
		RedirectURL: h.config.StatusRedirectURL,
	}

	if err := h.statusTemplate.Execute(w, data); err != nil {
		http.Error(w, "Error rendering status page", http.StatusInternalServerError)
		return
	}
}
