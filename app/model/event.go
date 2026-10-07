package model

import "time"

// Event merepresentasikan entitas event di database.
type Event struct {
	ID                  int       `json:"id"`
	OrganizerID         int       `json:"organizer_id"`
	CategoryID          int       `json:"category_id"`
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	Location            string    `json:"location"`
	Capacity            int       `json:"capacity"`
	Price               float64   `json:"price"`
	Status              string    `json:"status"` // draft, published, cancelled, finished
	StartsAt            time.Time `json:"starts_at"`
	EndsAt              time.Time `json:"ends_at"`
	RegistrationDeadline time.Time `json:"registration_deadline"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	// Relasi opsional saat di-query
	OrganizerName        *string `json:"organizer_name,omitempty"`
	CategoryName         *string `json:"category_name,omitempty"`
}

// CreateEventRequest memuat data untuk pembuatan event baru.
type CreateEventRequest struct {
	CategoryID           int       `json:"category_id" validate:"required,min=1"`
	Title                string    `json:"title" validate:"required,min=3,max=150"`
	Description          string    `json:"description" validate:"required"`
	Location             string    `json:"location" validate:"required,min=3,max=150"`
	Capacity             int       `json:"capacity" validate:"required,min=1"`
	Price                float64   `json:"price" validate:"gte=0"`
	StartsAt             time.Time `json:"starts_at" validate:"required"`
	EndsAt               time.Time `json:"ends_at" validate:"required"`
	RegistrationDeadline time.Time `json:"registration_deadline" validate:"required"`
}

// PutEventRequest memuat data untuk pembaruan penuh (PUT) event (wajib semua field).
type PutEventRequest struct {
	CategoryID           int       `json:"category_id" validate:"required,min=1"`
	Title                string    `json:"title" validate:"required,min=3,max=150"`
	Description          string    `json:"description" validate:"required"`
	Location             string    `json:"location" validate:"required,min=3,max=150"`
	Capacity             int       `json:"capacity" validate:"required,min=1"`
	Price                float64   `json:"price" validate:"gte=0"`
	StartsAt             time.Time `json:"starts_at" validate:"required"`
	EndsAt               time.Time `json:"ends_at" validate:"required"`
	RegistrationDeadline time.Time `json:"registration_deadline" validate:"required"`
}

// PatchEventRequest memuat data untuk pembaruan sebagian (PATCH) event (menggunakan pointer untuk omitnil).
type PatchEventRequest struct {
	CategoryID           *int       `json:"category_id,omitempty" validate:"omitempty,min=1"`
	Title                *string    `json:"title,omitempty" validate:"omitempty,min=3,max=150"`
	Description          *string    `json:"description,omitempty" validate:"omitempty"`
	Location             *string    `json:"location,omitempty" validate:"omitempty,min=3,max=150"`
	Capacity             *int       `json:"capacity,omitempty" validate:"omitempty,min=1"`
	Price                *float64   `json:"price,omitempty" validate:"omitempty,gte=0"`
	StartsAt             *time.Time `json:"starts_at,omitempty"`
	EndsAt               *time.Time `json:"ends_at,omitempty"`
	RegistrationDeadline *time.Time `json:"registration_deadline,omitempty"`
}

// UpdateEventStatusRequest memuat status baru event.
type UpdateEventStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=draft published cancelled finished"`
}

// EventCursorResponse merepresentasikan respons list event dengan cursor pagination.
type EventCursorResponse struct {
	Data       []Event `json:"data"`
	Pagination struct {
		Limit      int     `json:"limit"`
		NextCursor *string `json:"next_cursor"`
		HasMore    bool    `json:"has_more"`
	} `json:"pagination"`
}
