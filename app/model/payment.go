package model

import "time"

// entitas pembayaran registrasi event di database
type Payment struct {
	ID              int        `json:"id"`
	RegistrationID  int        `json:"registration_id"`
	Amount          int64      `json:"amount"`
	Method          *string    `json:"method,omitempty"`
	ReferenceNumber *string    `json:"reference_number,omitempty"`
	Status          string     `json:"status"` // unpaid, submitted, verified, rejected
	Note            string     `json:"note"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	VerifiedBy      *int       `json:"verified_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// referensi pembayaran yang dikirim peserta
type SubmitPaymentRequest struct {
	Method          string `json:"method" validate:"required,min=2,max=30"`
	ReferenceNumber string `json:"reference_number" validate:"required,min=3,max=60"`
}

//  verifikasi pembayaran oleh organizer
type VerifyPaymentRequest struct {
	Status string `json:"status" validate:"required,oneof=verified rejected"`
	Note   string `json:"note" validate:"max=200"`
}
