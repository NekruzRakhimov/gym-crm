package models

import "time"

type Client struct {
	ID         int       `db:"id" json:"id"`
	FullName   string    `db:"full_name" json:"full_name"`
	Phone      *string   `db:"phone" json:"phone"`
	PhotoPath  *string   `db:"photo_path" json:"photo_path"`
	CardNumber *string   `db:"card_number" json:"card_number"`
	IsActive   bool      `db:"is_active" json:"is_active"`
	Balance    float64   `db:"balance" json:"balance"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type ClientWithTariff struct {
	Client
	ActiveTariffName *string    `db:"active_tariff_name" json:"active_tariff_name"`
	ActiveTariffEnd  *time.Time `db:"active_tariff_end" json:"active_tariff_end"`
}

type CreateClientInput struct {
	FullName   string  `json:"full_name" binding:"required"`
	Phone      *string `json:"phone"`
	CardNumber *string `json:"card_number"`
}

type UpdateClientInput struct {
	FullName   string  `json:"full_name" binding:"required"`
	Phone      *string `json:"phone"`
	CardNumber *string `json:"card_number"`
}
