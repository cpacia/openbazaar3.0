package models

import "time"

// Event is a model that can be used to record the time of when
// events happened in the database.
type Event struct {
	Name string `gorm:"primary_key"`
	Time time.Time
}
