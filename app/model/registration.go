package model

import "time"

// entitas pendaftaran event di database
type Registration struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	EventID         int       `json:"event_id"`
	TicketTypeID    int       `json:"ticket_type_id"`
	Status          string    `json:"status"` // pending, confirmed, checked_in, cancelled
	PriceAtPurchase int64     `json:"price_at_purchase"`
	TicketCode      *string   `json:"ticket_code,omitempty"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty"`
	CheckedInAt     *time.Time `json:"checked_in_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relasi opsional saat di-query
	Username   *string `json:"username,omitempty"`
	Email      *string `json:"email,omitempty"`
	FullName   *string `json:"full_name,omitempty"`
	EventTitle *string `json:"event_title,omitempty"`
	TicketName *string `json:"ticket_name,omitempty"`
}

// data untuk mendaftar event (pilih jenis tiket)
type CreateRegistrationRequest struct {
	TicketTypeID int `json:"ticket_type_id" validate:"required,min=1"`
}

// respons list registrasi dengan cursor pagination
type RegistrationCursorResponse struct {
	Data       []Registration `json:"data"`
	Pagination struct {
		Limit      int     `json:"limit"`
		NextCursor *string `json:"next_cursor"`
		HasMore    bool    `json:"has_more"`
	} `json:"pagination"`
}
