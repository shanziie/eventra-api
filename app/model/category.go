package model

import "time"

// kategori event di database.
type Category struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Cpembuatan atau pembaruan kategori event.
type CategoryRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=80"`
	Description string `json:"description" validate:"max=255"`
}
