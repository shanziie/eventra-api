package service

import (
	"time"

	"eventra-api/helper"
)

// CanManageEvent memeriksa apakah user memiliki hak mengelola event (admin dengan event:update:any / event:delete:any atau pemilik event/organizer).
func CanManageEvent(currentUserID int, currentRole string, organizerID int, anyPermission string) bool {
	if currentRole == "admin" || helper.Can(currentRole, anyPermission) {
		return true
	}
	if currentRole == "organizer" && currentUserID == organizerID {
		return true
	}
	return false
}

// CanReadEvent memeriksa visibilitas event berdasarkan BR-E9:
// - participant: hanya melihat 'published' dan 'finished'
// - organizer: melihat 'published', 'finished', atau event miliknya sendiri
// - admin: melihat semua status
func CanReadEvent(currentRole string, eventStatus string, currentUserID int, organizerID int) bool {
	if currentRole == "admin" {
		return true
	}
	if currentRole == "organizer" {
		if eventStatus == "published" || eventStatus == "finished" || currentUserID == organizerID {
			return true
		}
		return false
	}
	// participant / guest / dll
	return eventStatus == "published" || eventStatus == "finished"
}

// ValidateEventStatusTransition memvalidasi transisi state machine event:
// - draft -> published
// - draft / published -> cancelled
// - published -> finished (hanya jika sekarang >= ends_at)
func ValidateEventStatusTransition(currentStatus, newStatus string, now time.Time, endsAt time.Time) error {
	if currentStatus == newStatus {
		return nil
	}

	switch currentStatus {
	case "draft":
		if newStatus == "published" || newStatus == "cancelled" {
			return nil
		}
	case "published":
		if newStatus == "cancelled" {
			return nil
		}
		if newStatus == "finished" {
			if now.Before(endsAt) {
				return helper.ConflictCode("INVALID_TRANSITION", "event belum selesai (belum mencapai waktu ends_at)")
			}
			return nil
		}
	case "cancelled", "finished":
		return helper.ConflictCode("INVALID_TRANSITION", "event dengan status final (cancelled/finished) tidak dapat diubah statusnya")
	}

	return helper.ConflictCode("INVALID_TRANSITION", "transisi status event tidak valid")
}

// ValidateEventModification memeriksa BR-E5: event yang punya registrasi aktif tidak boleh diubah waktu atau kapasitasnya.
func ValidateEventModification(status string, activeRegCount int) error {
	if activeRegCount > 0 {
		return helper.ConflictCode("HAS_ACTIVE_REGISTRATIONS", "event yang memiliki registrasi aktif tidak boleh diubah detail utamanya")
	}
	return nil
}

// ValidateEventDeletion memeriksa BR-E6: hanya event 'draft' yang boleh dihapus.
func ValidateEventDeletion(status string) error {
	if status != "draft" {
		return helper.ConflictCode("CANNOT_DELETE", "hanya event dengan status draft yang boleh dihapus")
	}
	return nil
}

// ValidateTicketTypeQuota memeriksa BR-E3: jumlah total kuota seluruh jenis tiket <= kapasitas event.
func ValidateTicketTypeQuota(eventCapacity, currentTotalQuota, addedQuota int) error {
	if currentTotalQuota+addedQuota > eventCapacity {
		return helper.Validation("validasi gagal", map[string]string{
			"quota": "total kuota jenis tiket melebihi kapasitas event",
		})
	}
	return nil
}
