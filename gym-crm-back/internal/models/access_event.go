package models

import "time"

type AccessEvent struct {
	ID            int64      `db:"id" json:"id"`
	ClientID      *int       `db:"client_id" json:"client_id"`
	TerminalID    *int       `db:"terminal_id" json:"terminal_id"`
	Direction     string     `db:"direction" json:"direction"`
	AuthMethod    *string    `db:"auth_method" json:"auth_method"`
	AccessGranted bool       `db:"access_granted" json:"access_granted"`
	DenyReason    *string    `db:"deny_reason" json:"deny_reason"`
	RawEvent      []byte     `db:"raw_event" json:"raw_event,omitempty"`
	EventTime     time.Time  `db:"event_time" json:"event_time"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

type AccessEventDetail struct {
	AccessEvent
	ClientName   *string `db:"client_name" json:"client_name"`
	ClientPhoto  *string `db:"client_photo" json:"client_photo"`
	TerminalName *string `db:"terminal_name" json:"terminal_name"`
}

type EventsFilter struct {
	From       string `form:"from"`
	To         string `form:"to"`
	ClientID   *int   `form:"client_id"`
	TerminalID *int   `form:"terminal_id"`
	Direction  string `form:"direction"`
	Granted    *bool  `form:"granted"`
	Page       int    `form:"page,default=1"`
	Limit      int    `form:"limit,default=20"`
}

type DashboardStats struct {
	InsideNow    int `json:"inside_now"`
	TodayEntries int `json:"today_entries"`
	TodayExits   int `json:"today_exits"`
	TodayDenied  int `json:"today_denied"`
}
