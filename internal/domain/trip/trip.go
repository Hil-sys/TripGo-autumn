package trip

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID             string
	UserID         string
	DriverID       string
	StartPoint     Point
	EndPoint       Point
	Price          int64
	Status         Status
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

// Finish переводит активную поездку в статус completed.
func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	return nil
}

func ValidateCreate(userID, driverID uuid.UUID, start, end Point, price int64) error {
	if userID == uuid.Nil {
		return ValidationError{Field: "user_id", Reason: "cannot be empty"}
	}
	if driverID == uuid.Nil {
		return ValidationError{Field: "driver_id", Reason: "cannot be empty"}
	}
	if price < 0 {
		return ValidationError{Field: "price", Reason: "cannot be negative"}
	}
	if start.Latitude < -90 || start.Latitude > 90 || start.Longitude < -180 || start.Longitude > 180 {
		return ValidationError{Field: "start_point", Reason: "coordinates out of range"}
	}
	if end.Latitude < -90 || end.Latitude > 90 || end.Longitude < -180 || end.Longitude > 180 {
		return ValidationError{Field: "end_point", Reason: "coordinates out of range"}
	}
	return nil
}
