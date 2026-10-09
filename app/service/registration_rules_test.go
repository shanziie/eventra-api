package service

import (
	"testing"
	"time"
)

func TestValidateRegistrationCreation(t *testing.T) {
	now := time.Now()
	deadline := now.Add(24 * time.Hour)
	pastDeadline := now.Add(-24 * time.Hour)

	// Valid: published and now before deadline
	if err := ValidateRegistrationCreation("published", deadline, now); err != nil {
		t.Errorf("expected nil error for valid registration creation, got %v", err)
	}

	// Invalid: event not published
	if err := ValidateRegistrationCreation("draft", deadline, now); err == nil {
		t.Errorf("expected error when event is not published")
	}

	// Invalid: now after registration deadline
	if err := ValidateRegistrationCreation("published", pastDeadline, now); err == nil {
		t.Errorf("expected error when registration deadline has passed")
	}
}

func TestValidateTicketTypeBelongsToEvent(t *testing.T) {
	if err := ValidateTicketTypeBelongsToEvent(1, 1); err != nil {
		t.Errorf("expected nil error when ticket type belongs to event")
	}
	if err := ValidateTicketTypeBelongsToEvent(2, 1); err == nil {
		t.Errorf("expected error when ticket type does not belong to event")
	}
}

func TestValidateCancellationWindow(t *testing.T) {
	now := time.Now()
	startsAt := now.Add(48 * time.Hour) // 48 hours in future (cancellation allowed up to 24h before startsAt)
	startsAtSoon := now.Add(12 * time.Hour) // 12 hours in future (within 24h window, cancellation not allowed)

	if err := ValidateCancellationWindow(startsAt, now); err != nil {
		t.Errorf("expected nil error when canceling well before 24h window, got %v", err)
	}

	if err := ValidateCancellationWindow(startsAtSoon, now); err == nil {
		t.Errorf("expected error when canceling within 24h before event starts")
	}
}

func TestValidateCheckInWindow(t *testing.T) {
	now := time.Now()
	startsAt := now.Add(1 * time.Hour) // starts in 1 hour (within 2-hour window before startsAt)
	endsAt := now.Add(4 * time.Hour)
	tooEarlyStarts := now.Add(5 * time.Hour) // check-in opens 2h before startsAt
	pastEndsAt := now.Add(-1 * time.Hour)

	// Valid check-in window
	if err := ValidateCheckInWindow(startsAt, endsAt, now); err != nil {
		t.Errorf("expected nil error within check-in window, got %v", err)
	}

	// Too early
	if err := ValidateCheckInWindow(tooEarlyStarts, endsAt, now); err == nil {
		t.Errorf("expected error when check-in is too early")
	}

	// Event ended
	if err := ValidateCheckInWindow(now.Add(-3 * time.Hour), pastEndsAt, now); err == nil {
		t.Errorf("expected error when event has ended")
	}
}

func TestCanAccessRegistration(t *testing.T) {
	// Reg owner can access
	if !CanAccessRegistration(1, "participant", 1, 2, "registration:read:any") {
		t.Errorf("expected registration owner to access registration")
	}

	// Organizer owner can access
	if !CanAccessRegistration(2, "organizer", 1, 2, "registration:read:any") {
		t.Errorf("expected organizer owner to access registration")
	}

	// Admin can access
	if !CanAccessRegistration(99, "admin", 1, 2, "registration:read:any") {
		t.Errorf("expected admin to access registration")
	}

	// Unauthorized user cannot access
	if CanAccessRegistration(3, "participant", 1, 2, "registration:read:any") {
		t.Errorf("expected unauthorized participant not to access registration")
	}
}
