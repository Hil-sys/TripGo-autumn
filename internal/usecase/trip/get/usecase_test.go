package get_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
	usecase "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/get"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	tripID := uuid.New().String()

	tests := []struct {
		name          string
		id            string
		setupMocks    func(r *MocktripRepository)
		expectedError error
	}{
		{
			name: "success: trip found",
			id:   tripID,
			setupMocks: func(r *MocktripRepository) {
				r.EXPECT().GetByID(gomock.Any(), tripID).Return(model.Trip{
					ID:     tripID,
					Status: string(domain.StatusActive),
				}, nil)
			},
			expectedError: nil,
		},
		{
			name: "validation failure: not a uuid",
			id:   "invalid-id",
			setupMocks: func(r *MocktripRepository) {},
			expectedError: domain.ValidationError{Field: "id", Reason: "must be a valid UUID"},
		},
		{
			name: "not found",
			id:   tripID,
			setupMocks: func(r *MocktripRepository) {
				r.EXPECT().GetByID(gomock.Any(), tripID).Return(model.Trip{}, domain.ErrTripNotFound)
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
			tt.setupMocks(repoMock)

			uc := usecase.New(repoMock)
			res, err := uc.Execute(context.Background(), tt.id)

			if tt.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tripID, res.ID)
			}
		})
	}
}
