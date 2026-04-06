package models

import "time"

type Terminal struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	IP        string    `db:"ip" json:"ip"`
	Port      int       `db:"port" json:"port"`
	Username  string    `db:"username" json:"username"`
	Password  string    `db:"password" json:"password"`
	Direction string    `db:"direction" json:"direction"`
	Active    bool      `db:"active" json:"active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type CreateTerminalInput struct {
	Name      string `json:"name" binding:"required"`
	IP        string `json:"ip" binding:"required"`
	Port      int    `json:"port"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	Direction string `json:"direction" binding:"required,oneof=entry exit"`
}

type UpdateTerminalInput struct {
	Name      string `json:"name" binding:"required"`
	IP        string `json:"ip" binding:"required"`
	Port      int    `json:"port"`
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password"` // optional — empty means keep existing
	Direction string `json:"direction" binding:"required,oneof=entry exit"`
}
