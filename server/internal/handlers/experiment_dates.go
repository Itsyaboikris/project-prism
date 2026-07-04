package handlers

import (
	"fmt"
	"time"
)

func validateExperimentDates(startDate, endDate *time.Time) error {
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		return fmt.Errorf("end date must be on or after start date")
	}

	return nil
}
