package models

import "time"

type ClientTariff struct {
	ID          int       `db:"id" json:"id"`
	ClientID    int       `db:"client_id" json:"client_id"`
	TariffID    int       `db:"tariff_id" json:"tariff_id"`
	StartDate   time.Time `db:"start_date" json:"start_date"`
	EndDate     time.Time `db:"end_date" json:"end_date"`
	PaidAmount  *float64  `db:"paid_amount" json:"paid_amount"`
	PaymentNote *string   `db:"payment_note" json:"payment_note"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type ClientTariffDetail struct {
	ClientTariff
	TariffName      string  `db:"tariff_name" json:"tariff_name"`
	DurationDays    int     `db:"duration_days" json:"duration_days"`
	MaxVisitDays    *int    `db:"max_visit_days" json:"max_visit_days"`
	ScheduleDays    string  `db:"schedule_days" json:"schedule_days"`
	TimeFrom        *string `db:"time_from" json:"time_from"`
	TimeTo          *string `db:"time_to" json:"time_to"`
}

type AssignTariffInput struct {
	TariffID  int    `json:"tariff_id" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
}
