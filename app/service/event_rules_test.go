package service

import (
	"testing"
	"time"
)

func TestCanManageEvent(t *testing.T) {
	// Admin can manage any event
	if !CanManageEvent(1, "admin", 2, "event:update:any") {
		t.Errorf("expected admin to be able to manage event")
	}

	// Organizer owner can manage own event
	if !CanManageEvent(2, "organizer", 2, "event:update:any") {
		t.Errorf("expected organizer owner to be able to manage event")
	}

	// Organizer non-owner without any permission cannot manage
	if CanManageEvent(3, "organizer", 2, "event:update:any") {
		t.Errorf("expected organizer non-owner not to manage event without permission")
	}

	// Participant cannot manage event
	if CanManageEvent(1, "participant", 2, "event:update:any") {
		t.Errorf("expected participant not to manage event")
	}
}

func TestCanReadEvent(t *testing.T) {
	now := time.Now()
	// Admin can read any event status
	if !CanReadEvent("admin", "draft", 1, 2) {
		t.Errorf("admin should read draft event")
	}

	// Organizer can read own draft event
	if !CanReadEvent("organizer", "draft", 2, 2) {
		t.Errorf("organizer should read own draft event")
	}

	// Organizer cannot read other's draft event
	if CanReadEvent("organizer", "draft", 3, 2) {
		t.Errorf("organizer should not read other's draft event")
	}

	// Participant can read published event
	if !CanReadEvent("participant", "published", 1, 2) {
		t.Errorf("participant should read published event")
	}

	// Participant cannot read draft event
	if CanReadEvent("participant", "draft", 1, 2) {
		t.Errorf("participant should not read draft event")
	}

	_ = now
}

func TestValidateEventStatusTransition(t *testing.T) {
	now := time.Now()
	future := now.Add(2 * time.Hour)
	past := now.Add(-2 * time.Hour)

	// draft -> published (valid)
	if err := ValidateEventStatusTransition("draft", "published", now, future); err != nil {
		t.Errorf("expected nil error for draft -> published, got %v", err)
	}

	// published -> finished after endsAt (valid)
	if err := ValidateEventStatusTransition("published", "finished", now, past); err != nil {
		t.Errorf("expected nil error for published -> finished after endsAt, got %v", err)
	}

	// published -> finished before endsAt (invalid)
	if err := ValidateEventStatusTransition("published", "finished", now, future); err == nil {
		t.Errorf("expected error for published -> finished before endsAt")
	}

	// cancelled -> published (invalid)
	if err := ValidateEventStatusTransition("cancelled", "published", now, future); err == nil {
		t.Errorf("expected error for cancelled -> published")
	}
}

func TestValidateEventModification(t *testing.T) {
	if err := ValidateEventModification("published", 0); err != nil {
		t.Errorf("expected nil error when 0 active registrations")
	}
	if err := ValidateEventModification("published", 1); err == nil {
		t.Errorf("expected error when active registrations > 0")
	}
}

func TestValidateEventDeletion(t *testing.T) {
	if err := ValidateEventDeletion("draft"); err != nil {
		t.Errorf("expected nil error for draft event deletion")
	}
	if err := ValidateEventDeletion("published"); err == nil {
		t.Errorf("expected error for published event deletion")
	}
}

func TestValidateTicketTypeQuota(t *testing.T) {
	if err := ValidateTicketTypeQuota(100, 50, 30); err != nil {
		t.Errorf("expected nil error when total quota <= capacity")
	}
	if err := ValidateTicketTypeQuota(100, 80, 30); err == nil {
		t.Errorf("expected error when total quota exceeds capacity")
	}
}
