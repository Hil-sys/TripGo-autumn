package domainerror

import (
	"errors"
	"net/http"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/generated"
)

func Problem(err error, instance string) (int, api.Problem) {
	var valErr domain.ValidationError
	if errors.As(err, &valErr) {
		status := http.StatusBadRequest
		code := "invalid_request"
		return status, api.Problem{
			Type:     "https://tripgo.example/problems/invalid-request",
			Title:    "Invalid Request",
			Status:   status,
			Detail:   valErr.Error(),
			Code:     &code,
			Instance: &instance,
		}
	}

	switch {
	case errors.Is(err, domain.ErrTripNotFound):
		status := http.StatusNotFound
		code := "trip_not_found"
		return status, api.Problem{
			Type:     "https://tripgo.example/problems/trip-not-found",
			Title:    "Trip Not Found",
			Status:   status,
			Detail:   err.Error(),
			Code:     &code,
			Instance: &instance,
		}

	case errors.Is(err, domain.ErrDriverBusy):
		status := http.StatusConflict
		code := "driver_busy"
		return status, api.Problem{
			Type:     "https://tripgo.example/problems/driver-busy",
			Title:    "Driver Busy",
			Status:   status,
			Detail:   err.Error(),
			Code:     &code,
			Instance: &instance,
		}

	case errors.Is(err, domain.ErrTripCompleted):
		status := http.StatusConflict
		code := "trip_completed"
		return status, api.Problem{
			Type:     "https://tripgo.example/problems/trip-completed",
			Title:    "Trip Already Completed",
			Status:   status,
			Detail:   err.Error(),
			Code:     &code,
			Instance: &instance,
		}

	default:
		status := http.StatusInternalServerError
		code := "internal_error"
		return status, api.Problem{
			Type:     "https://tripgo.example/problems/internal-error",
			Title:    "Internal Server Error",
			Status:   status,
			Detail:   "An unexpected error occurred",
			Code:     &code,
			Instance: &instance,
		}
	}
}
