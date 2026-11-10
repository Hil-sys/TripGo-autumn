package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
	usecase "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/create"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	validUserID := uuid.New().String()
	validDriverID := uuid.New().String()

	tests := []struct {
		name          string
		input         usecase.Input
		setupMocks    func(r *MocktripRepository, tm *MocktxManager)
		expectedError error
	}{
		{
			name: "success: trip created atomically",
			input: usecase.Input{
				UserID:     validUserID,
				DriverID:   validDriverID,
				StartPoint: model.Point{Latitude: 59.9398, Longitude: 30.3146},
				EndPoint:   model.Point{Latitude: 59.9290, Longitude: 30.3626},
				Price:      1450,
			},
			setupMocks: func(r *MocktripRepository, tm *MocktxManager) {
				tm.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					})

				r.EXPECT().
					SaveTrip(gomock.Any(), gomock.Any()).
					Return(nil)

				r.EXPECT().
					SaveStatusHistory(gomock.Any(), gomock.Any(), nil, string(domain.StatusActive)).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "validation failure: invalid user id",
			input: usecase.Input{
				UserID:   "invalid-uuid",
				DriverID: validDriverID,
			},
			setupMocks:    func(r *MocktripRepository, tm *MocktxManager) {},
			expectedError: domain.ValidationError{Field: "user_id", Reason: "must be a valid UUID"},
		},
		{
			name: "conflict: driver already has an active trip",
			input: usecase.Input{
				UserID:     validUserID,
				DriverID:   validDriverID,
				StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
				EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
				Price:      100,
			},
			setupMocks: func(r *MocktripRepository, tm *MocktxManager) {
				tm.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					})

				r.EXPECT().
					SaveTrip(gomock.Any(), gomock.Any()).
					Return(domain.ErrDriverBusy)
			},
			expectedError: domain.ErrDriverBusy,
		},
		{
			name: "rollback on history failure",
			input: usecase.Input{
				UserID:     validUserID,
				DriverID:   validDriverID,
				StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
				EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
				Price:      100,
			},
			setupMocks: func(r *MocktripRepository, tm *MocktxManager) {
				tm.EXPECT().
					Do(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						return fn(ctx)
					})

				r.EXPECT().SaveTrip(gomock.Any(), gomock.Any()).Return(nil)
				r.EXPECT().SaveStatusHistory(gomock.Any(), gomock.Any(), nil, string(domain.StatusActive)).
					Return(errors.New("db error"))
			},
			expectedError: errors.New("save status history: db error"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repoMock := NewMocktripRepository(ctrl)
			txMock := NewMocktxManager(ctrl)

			tt.setupMocks(repoMock, txMock)

			uc := usecase.New(repoMock, txMock)
			result, err := uc.Execute(context.Background(), tt.input)

			if tt.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tt.expectedError, domain.ErrDriverBusy) {
					assert.True(t, errors.Is(err, domain.ErrDriverBusy))
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result.ID)
				assert.Equal(t, string(domain.StatusActive), result.Status)
			}
		})
	}
}
