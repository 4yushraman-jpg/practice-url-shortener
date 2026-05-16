package models

import "time"

type URL struct {
	ID        int       `json:"id"`
	URL       string    `json:"url"`
	ShortCode string    `json:"short_code"`
	CreatedAt time.Time `json:"created_at"`
}
