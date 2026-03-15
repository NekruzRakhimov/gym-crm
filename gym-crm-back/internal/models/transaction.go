package models

import "time"

type Transaction struct {
	ID              int64      `db:"id" json:"id"`
	ClientID        int        `db:"client_id" json:"client_id"`
	Type            string     `db:"type" json:"type"` // deposit | payment
	Amount          float64    `db:"amount" json:"amount"`
	Description     *string    `db:"description" json:"description"`
	ClientTariffID  *int       `db:"client_tariff_id" json:"client_tariff_id"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

type DepositInput struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Description string  `json:"description"`
}

type FinanceStats struct {
	TotalRevenue    float64          `json:"total_revenue"`
	MonthlyRevenue  []MonthlyRevenue `json:"monthly_revenue"`
	TopTariffs      []TariffRevenue  `json:"top_tariffs"`
	TotalClients    int              `json:"total_clients"`
	ActiveClients   int              `json:"active_clients"`
}

type MonthlyRevenue struct {
	Month   string  `db:"month" json:"month"`
	Revenue float64 `db:"revenue" json:"revenue"`
}

type TariffRevenue struct {
	TariffName string  `db:"tariff_name" json:"tariff_name"`
	Count      int     `db:"count" json:"count"`
	Revenue    float64 `db:"revenue" json:"revenue"`
}
