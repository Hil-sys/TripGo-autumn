package trip

import (
	"time"

	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

type dbTripRow struct {
	ID             string
	UserID         string
	DriverID       string
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64
	Status         string
	StartedAt      time.Time
	FinishedAt     *time.Time
}

func toModelTrip(row dbTripRow) model.Trip {
	return model.Trip{
		ID:       row.ID,
		UserID:   row.UserID,
		DriverID: row.DriverID,
		StartPoint: model.Point{
			Latitude:  row.StartLatitude,
			Longitude: row.StartLongitude,
		},
		EndPoint: model.Point{
			Latitude:  row.EndLatitude,
			Longitude: row.EndLongitude,
		},
		Price:      row.Price,
		Status:     row.Status,
		StartedAt:  row.StartedAt,
		FinishedAt: row.FinishedAt,
	}
}
