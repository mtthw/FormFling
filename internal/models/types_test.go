package models

import (
	"encoding/json"
	"testing"
)

func TestFormData_JSON(t *testing.T) {
	t.Run("Marshal FormData", func(t *testing.T) {
		form := FormData{
			Name:    "John Doe",
			Email:   "john@example.com",
			Subject: "Test Subject",
			Message: "This is a test message",
			Phone:   "123-456-7890",
			Website: "https://example.com",
		}

		data, err := json.Marshal(form)
		if err != nil {
			t.Errorf("Failed to marshal FormData: %v", err)
		}

		expected := `{"name":"John Doe","email":"john@example.com","subject":"Test Subject","message":"This is a test message","phone":"123-456-7890","website":"https://example.com"}`
		if string(data) != expected {
			t.Errorf("Marshaled JSON doesn't match expected.\nGot: %s\nExpected: %s", string(data), expected)
		}
	})

	t.Run("Unmarshal FormData", func(t *testing.T) {
		jsonData := `{"name":"Jane Doe","email":"jane@example.com","subject":"Test","message":"Hello","phone":"","website":""}`

		var form FormData
		err := json.Unmarshal([]byte(jsonData), &form)
		if err != nil {
			t.Errorf("Failed to unmarshal FormData: %v", err)
		}

		if form.Name != "Jane Doe" {
			t.Errorf("Expected name 'Jane Doe', got %s", form.Name)
		}
		if form.Email != "jane@example.com" {
			t.Errorf("Expected email 'jane@example.com', got %s", form.Email)
		}
		if form.Subject != "Test" {
			t.Errorf("Expected subject 'Test', got %s", form.Subject)
		}
		if form.Message != "Hello" {
			t.Errorf("Expected message 'Hello', got %s", form.Message)
		}
		if form.Phone != "" {
			t.Errorf("Expected empty phone, got %s", form.Phone)
		}
		if form.Website != "" {
			t.Errorf("Expected empty website, got %s", form.Website)
		}
	})
}

func TestResponse_JSON(t *testing.T) {
	t.Run("Marshal successful response", func(t *testing.T) {
		response := Response{
			Status: "message sent",
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Errorf("Failed to marshal Response: %v", err)
		}

		expected := `{"status":"message sent"}`
		if string(data) != expected {
			t.Errorf("Marshaled JSON doesn't match expected.\nGot: %s\nExpected: %s", string(data), expected)
		}
	})

	t.Run("Marshal error response", func(t *testing.T) {
		response := Response{
			Status: "error",
			Error:  "validation failed",
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Errorf("Failed to marshal Response: %v", err)
		}

		expected := `{"status":"error","error":"validation failed"}`
		if string(data) != expected {
			t.Errorf("Marshaled JSON doesn't match expected.\nGot: %s\nExpected: %s", string(data), expected)
		}
	})

	t.Run("Unmarshal response", func(t *testing.T) {
		jsonData := `{"status":"error","error":"email not valid"}`

		var response Response
		err := json.Unmarshal([]byte(jsonData), &response)
		if err != nil {
			t.Errorf("Failed to unmarshal Response: %v", err)
		}

		if response.Status != "error" {
			t.Errorf("Expected status 'error', got %s", response.Status)
		}
		if response.Error != "email not valid" {
			t.Errorf("Expected error 'email not valid', got %s", response.Error)
		}
	})
}
