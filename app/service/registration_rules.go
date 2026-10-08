package service

import (
	"time"

	"eventra-api/helper"
)

// ValidateRegistrationCreation memeriksa BR-R1: event harus published dan sekarang <= registration_deadline.
func ValidateRegistrationCreation(eventStatus string, registrationDeadline time.Time, now time.Time) error {
	if eventStatus != "published" {
		return helper.ConflictCode("REGISTRATION_CLOSED", "pendaftaran hanya dibuka untuk event yang sudah dipublikasikan")
	}
	if now.After(registrationDeadline) {
		return helper.ConflictCode("REGISTRATION_CLOSED", "batas waktu pendaftaran event telah lewat")
	}
	return nil
}

// ValidateTicketTypeBelongsToEvent memeriksa BR-R2: jenis tiket harus milik event yang didaftar.
func ValidateTicketTypeBelongsToEvent(ticketTypeEventID, eventID int) error {
	if ticketTypeEventID != eventID {
		return helper.Validation("validasi gagal", map[string]string{
			"ticket_type_id": "jenis tiket tidak terdaftar pada event ini",
		})
	}
	return nil
}

// ValidateCancellationWindow memeriksa aturan pembatalan pendaftaran (maksimal H-24 sebelum event mulai).
func ValidateCancellationWindow(startsAt time.Time, now time.Time) error {
	deadline := startsAt.Add(-24 * time.Hour)
	if now.After(deadline) {
		return helper.ConflictCode("CANCELLATION_DEADLINE_PASSED", "pendaftaran hanya dapat dibatalkan maksimal H-24 sebelum event dimulai")
	}
	return nil
}

// ValidateCheckInWindow memeriksa apakah waktu check-in berada di dalam jendela acara (antara starts_at dan ends_at, atau toleransi tertentu).
func ValidateCheckInWindow(startsAt time.Time, endsAt time.Time, now time.Time) error {
	// Toleransi check-in: mulai 2 jam sebelum starts_at sampai ends_at
	windowStart := startsAt.Add(-2 * time.Hour)
	if now.Before(windowStart) {
		return helper.ConflictCode("CHECK_IN_NOT_OPEN", "jendela check-in belum dibuka")
	}
	if now.After(endsAt) {
		return helper.ConflictCode("CHECK_IN_CLOSED", "jendela check-in sudah ditutup (event telah berakhir)")
	}
	return nil
}

// CanAccessRegistration memeriksa apakah user berhak membaca/mengakses registrasi (pemilik registrasi, organizer event induk, atau admin dengan permission :any).
func CanAccessRegistration(currentUserID int, currentRole string, regUserID int, organizerID int, anyPermission string) bool {
	if currentUserID == regUserID || currentUserID == organizerID {
		return true
	}
	if currentRole == "admin" || helper.Can(currentRole, anyPermission) {
		return true
	}
	return false
}
