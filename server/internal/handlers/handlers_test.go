package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/db/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func testAlertHandler(t *testing.T) (*AlertHandler, *mocks.MockStoreService) {
	ctrl := gomock.NewController(t)
	mockStore := mocks.NewMockStoreService(ctrl)

	handler := &AlertHandler{
		store: mockStore,
	}

	return handler, mockStore
}

func testResortHandler(t *testing.T) (*ResortHandler, *mocks.MockStoreService) {
	ctrl := gomock.NewController(t)
	mockStore := mocks.NewMockStoreService(ctrl)

	handler := &ResortHandler{
		store: mockStore,
	}

	return handler, mockStore
}

func TestCreateAlert(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		requestBody     interface{}
		setupMock       func(*mocks.MockStoreService)
		expectedStatus  int
		expectedHeaders map[string]string
		expectedBody    map[string]string
	}{
		{
			name:   "Success",
			method: http.MethodPut,
			requestBody: CreateAlertRequest{
				Email:            "test@example.com",
				Phone:            "1234567890",
				NotificationDays: 3,
				MinSnowAmount:    5.0,
				ResortsUuids:     []string{"resort1", "resort2"},
			},
			setupMock: func(m *mocks.MockStoreService) {
				m.EXPECT().
					CreateUserWithAlerts(
						gomock.Any(),
						"test@example.com",
						"1234567890",
						5.0,
						int32(3),
						[]string{"resort1", "resort2"},
					).
					Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedHeaders: map[string]string{
				"Content-Type":           "application/json",
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"X-XSS-Protection":       "1; mode=block",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
			},
			expectedBody: map[string]string{
				"status":  "success",
				"message": "Alert created successfully",
			},
		},
		{
			name:   "Wrong HTTP Method",
			method: http.MethodGet,
			requestBody: CreateAlertRequest{
				Email:            "test@example.com",
				Phone:            "1234567890",
				NotificationDays: 3,
				MinSnowAmount:    5.0,
				ResortsUuids:     []string{"resort1"},
			},
			setupMock: func(m *mocks.MockStoreService) {
				// No calls expected
			},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:        "Invalid JSON Body",
			method:      http.MethodPut,
			requestBody: "this is not valid json",
			setupMock: func(m *mocks.MockStoreService) {
				// No calls expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Missing Required Fields - Empty Phone",
			method: http.MethodPut,
			requestBody: CreateAlertRequest{
				Email:            "test@example.com",
				Phone:            "", // Empty phone
				NotificationDays: 3,
				MinSnowAmount:    5.0,
				ResortsUuids:     []string{"resort1"},
			},
			setupMock: func(m *mocks.MockStoreService) {
				// No calls expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Missing Required Fields - Empty Resorts",
			method: http.MethodPut,
			requestBody: CreateAlertRequest{
				Email:            "test@example.com",
				Phone:            "1234567890",
				NotificationDays: 3,
				MinSnowAmount:    5.0,
				ResortsUuids:     []string{}, // Empty resorts
			},
			setupMock: func(m *mocks.MockStoreService) {
				// No calls expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Database Error",
			method: http.MethodPut,
			requestBody: CreateAlertRequest{
				Email:            "test@example.com",
				Phone:            "1234567890",
				NotificationDays: 3,
				MinSnowAmount:    5.0,
				ResortsUuids:     []string{"resort1"},
			},
			setupMock: func(m *mocks.MockStoreService) {
				m.EXPECT().
					CreateUserWithAlerts(
						gomock.Any(),
						"test@example.com",
						"1234567890",
						5.0,
						int32(3),
						[]string{"resort1"},
					).
					Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockStore := testAlertHandler(t)

			tt.setupMock(mockStore)

			var body bytes.Buffer
			switch v := tt.requestBody.(type) {
			case string:
				body.WriteString(v)
			default:
				err := json.NewEncoder(&body).Encode(tt.requestBody)
				require.NoError(t, err, "Failed to encode request body")
			}

			req, err := http.NewRequest(tt.method, "/api/alert", &body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			handler.CreateAlert(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, "Status code mismatch")

			if tt.expectedHeaders != nil {
				for key, value := range tt.expectedHeaders {
					assert.Equal(t, value, rr.Header().Get(key), "Header mismatch: %s", key)
				}
			}

			if tt.expectedStatus == http.StatusCreated {
				var response map[string]string
				err = json.NewDecoder(rr.Body).Decode(&response)
				require.NoError(t, err, "Failed to decode response body")
				assert.Equal(t, tt.expectedBody, response)
			}
		})
	}
}

func TestListAllResorts(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		setupMock       func(*mocks.MockStoreService)
		expectedStatus  int
		expectedHeaders map[string]string
		expectedResorts []dbgen.Resort
	}{
		{
			name:   "Success",
			method: http.MethodGet,
			setupMock: func(m *mocks.MockStoreService) {
				mockResorts := []dbgen.Resort{
					{
						ID:   1,
						Uuid: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
						Name: "Whistler Blackcomb",
					},
					{
						ID:   2,
						Uuid: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
						Name: "Vail",
					},
				}
				m.EXPECT().
					ListAllResorts(gomock.Any()).
					Return(mockResorts, nil)
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Type":           "application/json",
				"X-Content-Type-Options": "nosniff",
				"X-Frame-Options":        "DENY",
				"X-XSS-Protection":       "1; mode=block",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
			},
			expectedResorts: []dbgen.Resort{
				{
					ID:   1,
					Uuid: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
					Name: "Whistler Blackcomb",
				},
				{
					ID:   2,
					Uuid: uuid.MustParse("550e8400-e29b-41d4-a716-446655440002"),
					Name: "Vail",
				},
			},
		},
		{
			name:           "Wrong HTTP Method",
			method:         http.MethodPost,
			setupMock:      func(m *mocks.MockStoreService) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "Database Error",
			method: http.MethodGet,
			setupMock: func(m *mocks.MockStoreService) {
				m.EXPECT().
					ListAllResorts(gomock.Any()).
					Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, mockStore := testResortHandler(t)

			tt.setupMock(mockStore)

			req, err := http.NewRequest(tt.method, "/api/resorts", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			handler.ListAllResorts(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, "Status code mismatch")

			if tt.expectedHeaders != nil {
				for key, value := range tt.expectedHeaders {
					assert.Equal(t, value, rr.Header().Get(key), "Header mismatch: %s", key)
				}
			}

			if tt.expectedStatus == http.StatusOK && tt.expectedResorts != nil {
				var response []dbgen.Resort
				err = json.NewDecoder(rr.Body).Decode(&response)
				require.NoError(t, err, "Failed to decode response body")
				assert.Equal(t, tt.expectedResorts, response)
			}
		})
	}
}

func TestSetSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	setSecurityHeaders(w)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range expectedHeaders {
		assert.Equal(t, expectedValue, w.Header().Get(header))
	}
}
