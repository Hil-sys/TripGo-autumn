package get_test

import (
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
	gethandler "github.com/Hil-sys/TripGo-autumn/internal/http/trip/get"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tripID := uuid.New().String()

	tests := []struct {
		name           string
		tripID         string
		setupMocks     func(uc *MockusecaseInterface)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:   "success: 200 get trip",
			tripID: tripID,
			setupMocks: func(uc *MockusecaseInterface) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return(model.Trip{
					ID:         tripID,
					UserID:     uuid.New().String(),
					DriverID:   uuid.New().String(),
					Status:     string(domain.StatusActive),
					StartedAt:  time.Now().UTC(),
					FinishedAt: nil,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "not found: 404 trip not found",
			tripID: tripID,
			setupMocks: func(uc *MockusecaseInterface) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return(model.Trip{}, domain.ErrTripNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "trip_not_found",
		},
		{
			name:   "internal error: 500 without details in json",
			tripID: tripID,
			setupMocks: func(uc *MockusecaseInterface) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return(model.Trip{}, errors.New("query failed unexpectedly"))
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
			tt.setupMocks(ucMock)

			h := gethandler.NewHandler(ucMock, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tt.tripID, nil)
			w := httptest.NewRecorder()

			h.Handle(w, req, tt.tripID)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedCode != "" {
				assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
				var prob api.Problem
				err := json.NewDecoder(w.Body).Decode(&prob)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedCode, prob.Code)
				assert.Equal(t, tt.expectedStatus, int(prob.Status))
				if tt.expectedStatus == http.StatusInternalServerError {
					assert.NotContains(t, *prob.Detail, "query failed unexpectedly")
				}
			}
		})
	}
}
