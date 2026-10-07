package model

import "time"

// TicketType merepresentasikan jenis tiket suatu event di database.
type TicketType struct {
	ID        int       `json:"id"`
	EventID   int       `json:"event_id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Quota     int       `json:"quota"`
	Sold      int       `json:"sold"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateTicketTypeRequest memuat data untuk pembuatan jenis tiket baru (BR-T1).
type CreateTicketTypeRequest struct {
	Name  string  `json:"name" validate:"required,min=2,max=80"`
	Price float64 `json:"price" validate:"gte=0"`
	Quota int     `json:"quota" validate:"required,min=1"`
}

// PutTicketTypeRequest memuat data untuk pembaruan jenis tiket.
type PutTicketTypeRequest struct {
	Name  string  `json:"name" validate:"required,min=2,max=80"`
	Price float64 `json:"price" validate:"gte=0"`
	Quota int     `json:"quota" validate:"required,min=1"`
}
