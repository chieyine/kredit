package credit

import (
	"errors"
	"time"
)

// CollectionInstant preserves the existing quick-sale Lagos cutoff (23:59)
// while removing the browser's timezone from financially material terms.
// This helper does not change any already accepted agreement.
func CollectionInstant(dueDate string, graceHours int) (time.Time, error) {
	if graceHours < 0 || graceHours > 720 {
		return time.Time{}, errors.New("grace hours must be between 0 and 720")
	}
	location, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		return time.Time{}, err
	}
	date, err := time.ParseInLocation("2006-01-02", dueDate, location)
	if err != nil || len(dueDate) != 10 || date.Format("2006-01-02") != dueDate {
		return time.Time{}, errors.New("a valid payment date is required")
	}
	cutoff := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 0, 0, location)
	return cutoff.Add(time.Duration(graceHours) * time.Hour).UTC(), nil
}

// ExplicitCollectionInstant interprets an advanced-form datetime as Nigerian
// business time. No URL or browser timezone can change its meaning.
func ExplicitCollectionInstant(local string) (time.Time, error) {
	location, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		return time.Time{}, err
	}
	date, err := time.ParseInLocation("2006-01-02T15:04", local, location)
	if err != nil || len(local) != 16 || date.Format("2006-01-02T15:04") != local {
		return time.Time{}, errors.New("a valid Nigerian collection date and time is required")
	}
	return date.UTC(), nil
}
