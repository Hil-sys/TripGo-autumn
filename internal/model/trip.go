package model

import (
	"time"

	"github.com/google/uuid"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     Point
	EndPoint       Point
	Price          int64
	Status         string
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}
