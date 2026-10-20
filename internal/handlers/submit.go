package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"formfling/internal/config"
	"formfling/internal/models"
	"formfling/internal/services"
	"formfling/internal/utils"
)

type SubmitHandler struct {
	emailService services.EmailSender
}

func NewSubmitHandler(emailService services.EmailSender) *SubmitHandler {
	return &SubmitHandler{
		emailService: emailService,
	}
}

func (h *SubmitHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.handleError(w, r, "must be a post", http.StatusMethodNotAllowed)
		return
	}
	if r.ContentLength > config.MaxRequestBodyBytes {
		h.handleBodyTooLarge(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, config.MaxRequestBodyBytes)
	defer r.Body.Close()

	var formData models.FormData
	var err error

	// Parse data based on content type
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Parse JSON request body
		formData, err = h.parseJSONRequest(r)
		if err != nil {
			if isBodyTooLarge(err) {
				h.handleBodyTooLarge(w, r)
				return
			}
			h.handleError(w, r, "failed to parse JSON", http.StatusBadRequest)
			return
		}
	} else {
		// Parse form data (default)
		if strings.HasPrefix(contentType, "multipart/form-data") {
			err = r.ParseMultipartForm(config.MaxRequestBodyBytes)
			if r.MultipartForm != nil {
				defer r.MultipartForm.RemoveAll()
			}
		} else {
			err = r.ParseForm()
		}
		if err != nil {
			if isBodyTooLarge(err) {
				h.handleBodyTooLarge(w, r)
				return
			}
			h.handleError(w, r, "failed to parse form", http.StatusBadRequest)
			return
		}
		formData = models.FormData{
			Name:    utils.CleanString(r.FormValue("name")),
			Email:   utils.CleanString(r.FormValue("email")),
			Subject: utils.CleanString(r.FormValue("subject")),
			Message: utils.CleanString(r.FormValue("message")),
			Phone:   utils.CleanString(r.FormValue("phone")),
			Website: utils.CleanString(r.FormValue("website")),
		}
	}

	// Validate form
	if err := utils.ValidateForm(formData); err != nil {
		h.handleError(w, r, "server rejected", http.StatusBadRequest)
		return
	}

	// Get origin for email
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}

	// Send email
	if err := h.emailService.SendEmail(formData, origin); err != nil {
		log.Printf("Error sending email: %v", err)
		h.handleError(w, r, "failed to send email", http.StatusInternalServerError)
		return
	}

	h.handleSuccess(w, r)
}

func (h *SubmitHandler) parseJSONRequest(r *http.Request) (models.FormData, error) {
	var formData models.FormData

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return formData, fmt.Errorf("failed to read request body: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(body, &formData); err != nil {
		return formData, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Clean the data
	formData.Name = utils.CleanString(formData.Name)
	formData.Email = utils.CleanString(formData.Email)
	formData.Subject = utils.CleanString(formData.Subject)
	formData.Message = utils.CleanString(formData.Message)
	formData.Phone = utils.CleanString(formData.Phone)
	formData.Website = utils.CleanString(formData.Website)

	return formData, nil
}

func isBodyTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func (h *SubmitHandler) handleBodyTooLarge(w http.ResponseWriter, r *http.Request) {
	if h.isAjaxRequest(r) {
		h.handleError(w, r, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
}

func (h *SubmitHandler) handleSuccess(w http.ResponseWriter, r *http.Request) {
	// Check if this is an AJAX request (API mode)
	if h.isAjaxRequest(r) {
		w.Header().Set("Content-Type", "application/json")
		response := models.Response{Status: "message sent"}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle form submission with redirect
	http.Redirect(w, r, "/status?type=success", http.StatusSeeOther)
}

func (h *SubmitHandler) handleError(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int) {
	// Check if this is an AJAX request (API mode)
	if h.isAjaxRequest(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		response := models.Response{Status: "error", Error: errorMsg}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle form submission with redirect to error page
	http.Redirect(w, r, "/status?type=error", http.StatusSeeOther)
}

func (h *SubmitHandler) isAjaxRequest(r *http.Request) bool {
	// Check for common AJAX indicators
	return r.Header.Get("X-Requested-With") == "XMLHttpRequest" ||
		strings.Contains(r.Header.Get("Content-Type"), "application/json") ||
		strings.Contains(r.Header.Get("Accept"), "application/json")
}
