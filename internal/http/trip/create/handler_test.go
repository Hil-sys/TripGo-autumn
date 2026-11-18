package create_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/generated"
	createhandler "github.com/Hil-sys/TripGo-autumn/internal/http/trip/create"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

const codeInvalidRequest = "invalid_request"

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	validUserID := uuid.New().String()
	validDriverID := uuid.New().String()
	validTripID := uuid.New().String()

	tests := []struct {
		name               string
		body               string
		setupMocks         func(uc *MockusecaseInterface, metrics *MockmetricsRecorder)
		expectedStatus     int
		expectedCode       string
		expectLocation     bool
		checkDetailDetails bool
	}{
		{
			name: "success: 201 created",
			body: `{
				"user_id": "` + validUserID + `",
				"driver_id": "` + validDriverID + `",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 1450
			}`,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{
					ID:         validTripID,
					UserID:     validUserID,
					DriverID:   validDriverID,
					StartPoint: model.Point{Latitude: 59.9398, Longitude: 30.3146},
					EndPoint:   model.Point{Latitude: 59.9290, Longitude: 30.3626},
					Price:      1450,
					Status:     string(domain.StatusActive),
					StartedAt:  time.Now().UTC(),
				}, nil)
				metrics.EXPECT().TripCreated(gomock.Any())
			},
			expectedStatus: http.StatusCreated,
			expectLocation: true,
		},
		{
			name: "bad request: missing price (usecase must not be called)",
			body: `{
				"user_id": "` + validUserID + `",
				"driver_id": "` + validDriverID + `",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626}
			}`,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				// usecase must NOT be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   codeInvalidRequest,
		},
		{
			name: "bad request: price null (usecase must not be called)",
			body: `{
				"user_id": "` + validUserID + `",
				"driver_id": "` + validDriverID + `",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": null
			}`,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				// usecase must NOT be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   codeInvalidRequest,
		},
		{
			name: "bad request: missing start_point (usecase must not be called)",
			body: `{
				"user_id": "` + validUserID + `",
				"driver_id": "` + validDriverID + `",
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 1000
			}`,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				// usecase must NOT be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   codeInvalidRequest,
		},
		{
			name: "bad request: latitude is null (usecase must not be called)",
			body: `{
				"user_id": "` + validUserID + `",
				"driver_id": "` + validDriverID + `",
				"start_point": {"latitude": null, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 1000
			}`,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				// usecase must NOT be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   codeInvalidRequest,
		},
		{
			name: "bad request: unknown field (usecase must not be called)",
			body: `{
				"user_id": "` + validUserID + `",
				"driver_id": "` + validDriverID + `",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 1450,
				"extra_field": "disallowed"
			}`,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				// usecase must NOT be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   codeInvalidRequest,
		},
		{
			name: "bad request: trailing json after first object (usecase must not be called)",
			body: `{
				"user_id": "` + validUserID + `",
				"driver_id": "` + validDriverID + `",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 1450
			} {"extra": 1}`,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				// usecase must NOT be called
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   codeInvalidRequest,
		},
		{
			name: "conflict: driver busy -> 409",
			body: `{
				"user_id": "` + validUserID + `",
				"driver_id": "` + validDriverID + `",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 1450
			}`,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{}, domain.ErrDriverBusy)
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "driver_busy",
		},
		{
			name: "internal error: 500 without details in json",
			body: `{
				"user_id": "` + validUserID + `",
				"driver_id": "` + validDriverID + `",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 1450
			}`,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(model.Trip{}, errors.New("sensitive db connection lost"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ucMock := NewMockusecaseInterface(ctrl)
			metricsMock := NewMockmetricsRecorder(ctrl)
			tt.setupMocks(ucMock, metricsMock)

			h := createhandler.NewHandler(ucMock, metricsMock, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.Handle(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectLocation {
				assert.Equal(t, "/api/v1/trips/"+validTripID, w.Header().Get("Location"))
				var tripResp api.Trip
				err := json.NewDecoder(w.Body).Decode(&tripResp)
				require.NoError(t, err)
				assert.Equal(t, validTripID, tripResp.Id.String())
			}

			if tt.expectedCode != "" {
				assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
				var prob api.Problem
				err := json.NewDecoder(w.Body).Decode(&prob)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, prob.Code)
				assert.Equal(t, tt.expectedStatus, int(prob.Status))

				if tt.expectedStatus == http.StatusInternalServerError {
					// ensure sensitive error details are NOT in detail
					assert.NotContains(t, *prob.Detail, "sensitive db connection lost")
				}
			}
		})
	}
}
