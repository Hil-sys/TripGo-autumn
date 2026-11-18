package domainerror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	t.Parallel()

	instancePath := "/api/v1/trips/123"

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
		expectedTitle  string
		expectedType   string
		checkDetail    func(t *testing.T, detail *string)
	}{
		{
			name: "validation error -> 400 invalid_request",
			err: domain.ValidationError{
				Field:  "price",
				Reason: "cannot be negative",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "invalid_request",
			expectedTitle:  "Invalid Request",
			expectedType:   "https://tripgo.example/problems/invalid-request",
			checkDetail: func(t *testing.T, detail *string) {
				require.NotNil(t, detail)
				assert.Contains(t, *detail, `invalid field "price": cannot be negative`)
			},
		},
		{
			name:           "not found error -> 404 trip_not_found",
			err:            domain.ErrTripNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "trip_not_found",
			expectedTitle:  "Trip Not Found",
			expectedType:   "https://tripgo.example/problems/trip-not-found",
			checkDetail: func(t *testing.T, detail *string) {
				require.NotNil(t, detail)
				assert.Equal(t, domain.ErrTripNotFound.Error(), *detail)
			},
		},
		{
			name:           "driver busy -> 409 driver_busy",
			err:            domain.ErrDriverBusy,
			expectedStatus: http.StatusConflict,
			expectedCode:   "driver_busy",
			expectedTitle:  "Driver Busy",
			expectedType:   "https://tripgo.example/problems/driver-busy",
			checkDetail: func(t *testing.T, detail *string) {
				require.NotNil(t, detail)
				assert.Equal(t, domain.ErrDriverBusy.Error(), *detail)
			},
		},
		{
			name:           "trip completed -> 409 trip_completed",
			err:            domain.ErrTripCompleted,
			expectedStatus: http.StatusConflict,
			expectedCode:   "trip_completed",
			expectedTitle:  "Trip Already Completed",
			expectedType:   "https://tripgo.example/problems/trip-completed",
			checkDetail: func(t *testing.T, detail *string) {
				require.NotNil(t, detail)
				assert.Equal(t, domain.ErrTripCompleted.Error(), *detail)
			},
		},
		{
			name:           "wrapped domain error -> 404 trip_not_found",
			err:            fmt.Errorf("wrapped context: %w", domain.ErrTripNotFound),
			expectedStatus: http.StatusNotFound,
			expectedCode:   "trip_not_found",
			expectedTitle:  "Trip Not Found",
			expectedType:   "https://tripgo.example/problems/trip-not-found",
			checkDetail: func(t *testing.T, detail *string) {
				require.NotNil(t, detail)
				assert.Contains(t, *detail, domain.ErrTripNotFound.Error())
			},
		},
		{
			name:           "internal error -> 500 without details leak",
			err:            errors.New("fatal: database connection password leaked"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "internal_error",
			expectedTitle:  "Internal Server Error",
			expectedType:   "https://tripgo.example/problems/internal-error",
			checkDetail: func(t *testing.T, detail *string) {
				require.NotNil(t, detail)
				assert.Equal(t, "An unexpected error occurred", *detail)
				assert.NotContains(t, *detail, "password leaked")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, prob := domainerror.Problem(tt.err, instancePath)

			assert.Equal(t, tt.expectedStatus, status)
			assert.Equal(t, tt.expectedStatus, int(prob.Status))
			assert.Equal(t, tt.expectedCode, prob.Code)
			assert.Equal(t, tt.expectedTitle, prob.Title)
			assert.Equal(t, tt.expectedType, prob.Type)

			require.NotNil(t, prob.Instance)
			assert.Equal(t, instancePath, *prob.Instance)

			tt.checkDetail(t, prob.Detail)
		})
	}
}
