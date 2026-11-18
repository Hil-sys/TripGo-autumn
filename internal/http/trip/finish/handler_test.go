package finish_test

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
	finishhandler "github.com/Hil-sys/TripGo-autumn/internal/http/trip/finish"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tripID := uuid.New().String()
	now := time.Now().UTC()

	tests := []struct {
		name           string
		tripID         string
		setupMocks     func(uc *MockusecaseInterface, metrics *MockmetricsRecorder)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:   "success: 200 finished",
			tripID: tripID,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return(model.Trip{
					ID:         tripID,
					UserID:     uuid.New().String(),
					DriverID:   uuid.New().String(),
					Status:     string(domain.StatusCompleted),
					StartedAt:  now.Add(-time.Hour),
					FinishedAt: &now,
				}, nil)
				metrics.EXPECT().TripCompleted(gomock.Any())
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "conflict: trip already completed -> 409",
			tripID: tripID,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return(model.Trip{}, domain.ErrTripCompleted)
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   "trip_completed",
		},
		{
			name:   "not found: trip does not exist -> 404",
			tripID: tripID,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return(model.Trip{}, domain.ErrTripNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "trip_not_found",
		},
		{
			name:   "internal error: 500 without details in json",
			tripID: tripID,
			setupMocks: func(uc *MockusecaseInterface, metrics *MockmetricsRecorder) {
				uc.EXPECT().Execute(gomock.Any(), tripID).Return(model.Trip{}, errors.New("db dead"))
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

			h := finishhandler.NewHandler(ucMock, metricsMock, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tt.tripID+"/finish", nil)
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
					assert.NotContains(t, *prob.Detail, "db dead")
				}
			}
		})
	}
}
