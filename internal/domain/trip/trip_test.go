package trip_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
)

const (
	reasonCannotBeNegative    = "cannot be negative"
	reasonCoordinatesOutRange = "coordinates out of range"
	fieldPrice                = "price"
	fieldStartPoint           = "start_point"
	fieldEndPoint             = "end_point"
)

func TestTrip_Finish(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name          string
		initialStatus trip.Status
		at            time.Time
		expectedErr   error
		expectSuccess bool
	}{
		{
			name:          "success: finish active trip",
			initialStatus: trip.StatusActive,
			at:            now,
			expectedErr:   nil,
			expectSuccess: true,
		},
		{
			name:          "conflict: already completed trip",
			initialStatus: trip.StatusCompleted,
			at:            now,
			expectedErr:   trip.ErrTripCompleted,
			expectSuccess: false,
		},
		{
			name:          "conflict: unknown status",
			initialStatus: trip.Status("unknown"),
			at:            now,
			expectedErr:   trip.ErrTripCompleted,
			expectSuccess: false,
		},
		{
			name:          "conflict: empty status",
			initialStatus: trip.Status(""),
			at:            now,
			expectedErr:   trip.ErrTripCompleted,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := &trip.Trip{
				ID:     uuid.New().String(),
				Status: tt.initialStatus,
			}

			err := tr.Finish(tt.at)

			if tt.expectSuccess {
				require.NoError(t, err)
				assert.Equal(t, trip.StatusCompleted, tr.Status)
				require.NotNil(t, tr.FinishedAt)
				assert.Equal(t, tt.at, *tr.FinishedAt)
			} else {
				require.ErrorIs(t, err, tt.expectedErr)
				assert.Equal(t, tt.initialStatus, tr.Status)
				assert.Nil(t, tr.FinishedAt)
			}
		})
	}
}

func TestValidateCreate(t *testing.T) {
	t.Parallel()

	validUserID := uuid.New()
	validDriverID := uuid.New()
	validStart := trip.Point{Latitude: 59.9398, Longitude: 30.3146}
	validEnd := trip.Point{Latitude: 59.9290, Longitude: 30.3626}

	tests := []struct {
		name          string
		userID        uuid.UUID
		driverID      uuid.UUID
		start         trip.Point
		end           trip.Point
		price         int64
		expectedErr   error
		expectedField string
	}{
		{
			name:        "valid: standard values",
			userID:      validUserID,
			driverID:    validDriverID,
			start:       validStart,
			end:         validEnd,
			price:       1450,
			expectedErr: nil,
		},
		{
			name:        "valid: zero price allowed",
			userID:      validUserID,
			driverID:    validDriverID,
			start:       validStart,
			end:         validEnd,
			price:       0,
			expectedErr: nil,
		},
		{
			name:        "valid: boundary coordinates -90, -180 and 90, 180",
			userID:      validUserID,
			driverID:    validDriverID,
			start:       trip.Point{Latitude: -90, Longitude: -180},
			end:         trip.Point{Latitude: 90, Longitude: 180},
			price:       100,
			expectedErr: nil,
		},
		{
			name:          "invalid: nil user_id",
			userID:        uuid.Nil,
			driverID:      validDriverID,
			start:         validStart,
			end:           validEnd,
			price:         100,
			expectedErr:   trip.ValidationError{Field: "user_id", Reason: "cannot be empty"},
			expectedField: "user_id",
		},
		{
			name:          "invalid: nil driver_id",
			userID:        validUserID,
			driverID:      uuid.Nil,
			start:         validStart,
			end:           validEnd,
			price:         100,
			expectedErr:   trip.ValidationError{Field: "driver_id", Reason: "cannot be empty"},
			expectedField: "driver_id",
		},
		{
			name:          "invalid: negative price",
			userID:        validUserID,
			driverID:      validDriverID,
			start:         validStart,
			end:           validEnd,
			price:         -1,
			expectedErr:   trip.ValidationError{Field: fieldPrice, Reason: reasonCannotBeNegative},
			expectedField: fieldPrice,
		},
		{
			name:          "invalid: start latitude out of range (> 90)",
			userID:        validUserID,
			driverID:      validDriverID,
			start:         trip.Point{Latitude: 90.0001, Longitude: 30.0},
			end:           validEnd,
			price:         100,
			expectedErr:   trip.ValidationError{Field: fieldStartPoint, Reason: reasonCoordinatesOutRange},
			expectedField: fieldStartPoint,
		},
		{
			name:          "invalid: start latitude out of range (< -90)",
			userID:        validUserID,
			driverID:      validDriverID,
			start:         trip.Point{Latitude: -90.0001, Longitude: 30.0},
			end:           validEnd,
			price:         100,
			expectedErr:   trip.ValidationError{Field: fieldStartPoint, Reason: reasonCoordinatesOutRange},
			expectedField: fieldStartPoint,
		},
		{
			name:          "invalid: start longitude out of range (> 180)",
			userID:        validUserID,
			driverID:      validDriverID,
			start:         trip.Point{Latitude: 50.0, Longitude: 180.0001},
			end:           validEnd,
			price:         100,
			expectedErr:   trip.ValidationError{Field: fieldStartPoint, Reason: reasonCoordinatesOutRange},
			expectedField: fieldStartPoint,
		},
		{
			name:          "invalid: start longitude out of range (< -180)",
			userID:        validUserID,
			driverID:      validDriverID,
			start:         trip.Point{Latitude: 50.0, Longitude: -180.0001},
			end:           validEnd,
			price:         100,
			expectedErr:   trip.ValidationError{Field: fieldStartPoint, Reason: reasonCoordinatesOutRange},
			expectedField: fieldStartPoint,
		},
		{
			name:          "invalid: end latitude out of range (> 90)",
			userID:        validUserID,
			driverID:      validDriverID,
			start:         validStart,
			end:           trip.Point{Latitude: 90.1, Longitude: 30.0},
			price:         100,
			expectedErr:   trip.ValidationError{Field: fieldEndPoint, Reason: reasonCoordinatesOutRange},
			expectedField: fieldEndPoint,
		},
		{
			name:          "invalid: end longitude out of range (< -180)",
			userID:        validUserID,
			driverID:      validDriverID,
			start:         validStart,
			end:           trip.Point{Latitude: 50.0, Longitude: -180.1},
			price:         100,
			expectedErr:   trip.ValidationError{Field: fieldEndPoint, Reason: reasonCoordinatesOutRange},
			expectedField: fieldEndPoint,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := trip.ValidateCreate(tt.userID, tt.driverID, tt.start, tt.end, tt.price)

			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)

				var valErr trip.ValidationError
				require.ErrorAs(t, err, &valErr)
				assert.Equal(t, tt.expectedField, valErr.Field)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	err := trip.ValidationError{
		Field:  fieldPrice,
		Reason: reasonCannotBeNegative,
	}

	assert.Equal(t, `invalid field "price": cannot be negative`, err.Error())
}
