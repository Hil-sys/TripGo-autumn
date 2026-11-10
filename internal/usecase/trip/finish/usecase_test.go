package finish_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
	usecase "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/finish"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	tripID := uuid.New().String()

	tests := []struct {
		name          string
		id            string
		setupMocks    func(r *MocktripRepository, tm *MocktxManager)
		expectedError error
	}{
		{
			name: "success: finishes active trip",
			id:   tripID,
			setupMocks: func(r *MocktripRepository, tm *MocktxManager) {
				tm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				})

				r.EXPECT().GetByIDForUpdate(gomock.Any(), tripID).Return(model.Trip{
					ID:        tripID,
					Status:    string(domain.StatusActive),
					StartedAt: time.Now().UTC(),
				}, nil)

				r.EXPECT().UpdateFinishedStatus(gomock.Any(), tripID, gomock.Any()).Return(nil)
				r.EXPECT().SaveStatusHistory(gomock.Any(), tripID, gomock.Any(), string(domain.StatusCompleted)).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "conflict: already completed trip",
			id:   tripID,
			setupMocks: func(r *MocktripRepository, tm *MocktxManager) {
				tm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				})

				r.EXPECT().GetByIDForUpdate(gomock.Any(), tripID).Return(model.Trip{
					ID:     tripID,
					Status: string(domain.StatusCompleted),
				}, nil)
			},
			expectedError: domain.ErrTripCompleted,
		},
		{
			name: "not found: trip does not exist",
			id:   tripID,
			setupMocks: func(r *MocktripRepository, tm *MocktxManager) {
				tm.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				})

				r.EXPECT().GetByIDForUpdate(gomock.Any(), tripID).Return(model.Trip{}, domain.ErrTripNotFound)
			},
			expectedError: domain.ErrTripNotFound,
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
			res, err := uc.Execute(context.Background(), tt.id)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, string(domain.StatusCompleted), res.Status)
				assert.NotNil(t, res.FinishedAt)
			}
		})
	}
}
