package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactHandler_HandleContact(t *testing.T) {
	handler := &ContactHandler{}

	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		expectedStatus int
		expectedError  *ErrorResponse
	}{
		{
			name:   "Success",
			method: http.MethodPost,
			requestBody: ContactRequest{
				Name:    "John Doe",
				Email:   "john@example.com",
				Message: "I have a question about your service.",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Wrong HTTP Method",
			method: http.MethodGet,
			requestBody: ContactRequest{
				Name:    "John Doe",
				Email:   "john@example.com",
				Message: "Test message",
			},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError: &ErrorResponse{
				Error:   "METHOD_NOT_ALLOWED",
				Message: "Method not allowed",
			},
		},
		{
			name:           "Invalid JSON Body",
			method:         http.MethodPost,
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError: &ErrorResponse{
				Error:   "INVALID_REQUEST",
				Message: "Invalid request body",
			},
		},
		{
			name:   "Missing Name",
			method: http.MethodPost,
			requestBody: ContactRequest{
				Name:    "",
				Email:   "john@example.com",
				Message: "Test message",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &ErrorResponse{
				Error:   "MISSING_NAME",
				Message: "Name is required",
			},
		},
		{
			name:   "Missing Email",
			method: http.MethodPost,
			requestBody: ContactRequest{
				Name:    "John Doe",
				Email:   "",
				Message: "Test message",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &ErrorResponse{
				Error:   "MISSING_EMAIL",
				Message: "Email is required",
			},
		},
		{
			name:   "Missing Message",
			method: http.MethodPost,
			requestBody: ContactRequest{
				Name:    "John Doe",
				Email:   "john@example.com",
				Message: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &ErrorResponse{
				Error:   "MISSING_MESSAGE",
				Message: "Message is required",
			},
		},
		{
			name:   "Invalid Email Format",
			method: http.MethodPost,
			requestBody: ContactRequest{
				Name:    "John Doe",
				Email:   "notanemail",
				Message: "Test message",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &ErrorResponse{
				Error:   "INVALID_EMAIL",
				Message: "Invalid email address",
			},
		},
		{
			name:   "Whitespace Only Fields",
			method: http.MethodPost,
			requestBody: ContactRequest{
				Name:    "   ",
				Email:   "john@example.com",
				Message: "Test message",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError: &ErrorResponse{
				Error:   "MISSING_NAME",
				Message: "Name is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			switch v := tt.requestBody.(type) {
			case string:
				body.WriteString(v)
			default:
				err := json.NewEncoder(&body).Encode(tt.requestBody)
				require.NoError(t, err, "Failed to encode request body")
			}

			req := httptest.NewRequest(tt.method, "/api/contact", &body)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.HandleContact(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, "Status code mismatch")

			if tt.expectedStatus == http.StatusOK {
				var response map[string]string
				err := json.NewDecoder(rr.Body).Decode(&response)
				require.NoError(t, err, "Failed to decode response body")
				assert.Equal(t, "success", response["status"])
				assert.NotEmpty(t, response["message"])
			} else if tt.expectedError != nil {
				var errorResponse ErrorResponse
				err := json.NewDecoder(rr.Body).Decode(&errorResponse)
				require.NoError(t, err, "Failed to decode error response body")
				assert.Equal(t, *tt.expectedError, errorResponse)
			}
		})
	}
}

func TestContactHandler_SecurityHeaders(t *testing.T) {
	handler := &ContactHandler{}

	requestBody := ContactRequest{
		Name:    "John Doe",
		Email:   "john@example.com",
		Message: "Test message",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.HandleContact(rr, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range expectedHeaders {
		assert.Equal(t, expectedValue, rr.Header().Get(header), "Header mismatch: %s", header)
	}
}
