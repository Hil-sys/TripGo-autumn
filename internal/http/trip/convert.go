package trip

import (
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/Hil-sys/TripGo-autumn/internal/generated"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

func ToAPI(t model.Trip) api.Trip {
	tripUUID, _ := uuid.Parse(t.ID)
	userID, _ := uuid.Parse(t.UserID)
	driverUUID, _ := uuid.Parse(t.DriverID)

	status := api.TripStatus(t.Status)

	return api.Trip{
		Id:       openapi_types.UUID(tripUUID),
		UserId:   openapi_types.UUID(userID),
		DriverId: openapi_types.UUID(driverUUID),
		StartPoint: api.Coordinates{
			Latitude:  t.StartPoint.Latitude,
			Longitude: t.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  t.EndPoint.Latitude,
			Longitude: t.EndPoint.Longitude,
		},
		Price:      t.Price,
		Status:     status,
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}