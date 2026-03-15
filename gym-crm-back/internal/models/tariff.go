package models

import "time"

type Tariff struct {
	ID              int       `db:"id" json:"id"`
	Name            string    `db:"name" json:"name"`
	DurationDays    int       `db:"duration_days" json:"duration_days"`
	MaxVisitsPerDay *int      `db:"max_visits_per_day" json:"max_visits_per_day"`
	Price           float64   `db:"price" json:"price"`
	Active          bool      `db:"active" json:"active"`
	ScheduleDays    string    `db:"schedule_days" json:"schedule_days"` // all|weekdays|weekends|even|odd
	TimeFrom        *string   `db:"time_from" json:"time_from"`         // "HH:MM" or nil
	TimeTo          *string   `db:"time_to" json:"time_to"`             // "HH:MM" or nil
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type CreateTariffInput struct {
	Name            string   `json:"name" binding:"required"`
	DurationDays    int      `json:"duration_days" binding:"required,min=1"`
	MaxVisitsPerDay *int     `json:"max_visits_per_day"`
	Price           float64  `json:"price" binding:"required,min=0"`
	ScheduleDays    string   `json:"schedule_days"`
	TimeFrom        *string  `json:"time_from"`
	TimeTo          *string  `json:"time_to"`
}
