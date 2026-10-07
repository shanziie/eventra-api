package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"eventra-api/app/model"
	"eventra-api/app/repository"
	"eventra-api/helper"

	"github.com/gofiber/fiber/v2"
)

type EventService struct {
	eventRepo      repository.EventRepository
	ticketTypeRepo repository.TicketTypeRepository
}

func NewEventService(eventRepo repository.EventRepository, ticketTypeRepo repository.TicketTypeRepository) *EventService {
	return &EventService{
		eventRepo:      eventRepo,
		ticketTypeRepo: ticketTypeRepo,
	}
}

// #17 GET /events
func (s *EventService) GetEvents(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	limit := 10
	if l := c.Query("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err == nil && v >= 1 && v <= 100 {
			limit = v
		}
	}

	var cursorTime *time.Time
	var cursorID *int
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		parts := strings.Split(cursorStr, "|")
		if len(parts) == 2 {
			if nano, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
				t := time.Unix(0, nano)
				cursorTime = &t
			}
			if id, err := strconv.Atoi(parts[1]); err == nil {
				cursorID = &id
			}
		}
	}

	search := c.Query("search")
	var categoryID *int
	if catStr := c.Query("category_id"); catStr != "" {
		if id, err := strconv.Atoi(catStr); err == nil {
			categoryID = &id
		}
	}
	status := c.Query("status")

	viewerRole := ""
	if r, err := helper.CurrentUserRole(c); err == nil {
		viewerRole = r
	}
	viewerID := 0
	if id, err := helper.CurrentUserID(c); err == nil {
		viewerID = id
	}

	events, nextCursor, hasMore, err := s.eventRepo.FindAll(ctx, limit, cursorTime, cursorID, search, categoryID, status, viewerRole, viewerID)
	if err != nil {
		return helper.Internal(err)
	}

	res := model.EventCursorResponse{
		Data: events,
	}
	res.Pagination.Limit = limit
	res.Pagination.NextCursor = nextCursor
	res.Pagination.HasMore = hasMore

	return helper.Success(c, "daftar event ditemukan", res)
}

// #18 GET /events/:id
func (s *EventService) GetEventByID(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	viewerRole := ""
	if r, err := helper.CurrentUserRole(c); err == nil {
		viewerRole = r
	}
	viewerID := 0
	if idVal, err := helper.CurrentUserID(c); err == nil {
		viewerID = idVal
	}

	if !CanReadEvent(viewerRole, event.Status, viewerID, event.OrganizerID) {
		return helper.NotFound("event tidak ditemukan") // BR-E9: event yang tidak terlihat dijawab 404
	}

	return helper.Success(c, "event ditemukan", event)
}

// #19 POST /events
func (s *EventService) CreateEvent(c *fiber.Ctx) error {
	var req model.CreateEventRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	if req.EndsAt.Before(req.StartsAt) || req.EndsAt.Equal(req.StartsAt) {
		return helper.Validation("validasi gagal", map[string]string{"ends_at": "waktu selesai harus setelah waktu mulai"})
	}
	if req.RegistrationDeadline.After(req.StartsAt) {
		return helper.Validation("validasi gagal", map[string]string{"registration_deadline": "batas pendaftaran tidak boleh setelah waktu mulai event"})
	}

	organizerID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	created, err := s.eventRepo.Create(ctx, organizerID, &req)
	if err != nil {
		return helper.Internal(err)
	}

	location := fmt.Sprintf("/api/v1/events/%d", created.ID)
	return helper.Created(c, location, "event berhasil dibuat", created)
}

// #20 PUT /events/:id
func (s *EventService) UpdateEvent(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	var req model.PutEventRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	if req.EndsAt.Before(req.StartsAt) || req.EndsAt.Equal(req.StartsAt) {
		return helper.Validation("validasi gagal", map[string]string{"ends_at": "waktu selesai harus setelah waktu mulai"})
	}
	if req.RegistrationDeadline.After(req.StartsAt) {
		return helper.Validation("validasi gagal", map[string]string{"registration_deadline": "batas pendaftaran tidak boleh setelah waktu mulai event"})
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	existing, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	viewerID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	viewerRole, _ := helper.CurrentUserRole(c)

	if !CanManageEvent(viewerID, viewerRole, existing.OrganizerID, "event:update:any") {
		return helper.Forbidden("tidak memiliki akses untuk mengubah event ini")
	}

	// BR-E5: Event dengan registrasi aktif tidak boleh diubah starts_at, ends_at, capacity
	activeRegCount, err := s.eventRepo.CountActiveRegistrations(ctx, id)
	if err == nil && activeRegCount > 0 {
		if existing.Capacity != req.Capacity || !existing.StartsAt.Equal(req.StartsAt) || !existing.EndsAt.Equal(req.EndsAt) {
			return helper.ConflictCode("HAS_ACTIVE_REGISTRATIONS", "event dengan registrasi aktif tidak boleh diubah kapasitas atau waktunya")
		}
	}

	updated, err := s.eventRepo.Update(ctx, id, &req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, "event berhasil diperbarui", updated)
}

// #21 PATCH /events/:id
func (s *EventService) PatchEvent(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	var req model.PatchEventRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	existing, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	viewerID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	viewerRole, _ := helper.CurrentUserRole(c)

	if !CanManageEvent(viewerID, viewerRole, existing.OrganizerID, "event:update:any") {
		return helper.Forbidden("tidak memiliki akses untuk mengubah event ini")
	}

	fields := make(map[string]any)
	if req.CategoryID != nil {
		fields["category_id"] = *req.CategoryID
	}
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Location != nil {
		fields["location"] = *req.Location
	}
	if req.Price != nil {
		fields["price"] = *req.Price
	}
	if req.RegistrationDeadline != nil {
		fields["registration_deadline"] = *req.RegistrationDeadline
	}

	// Cek BR-E5 jika capacity atau waktu diubah dan ada registrasi aktif
	needsCheckBR_E5 := (req.Capacity != nil && *req.Capacity != existing.Capacity) ||
		(req.StartsAt != nil && !req.StartsAt.Equal(existing.StartsAt)) ||
		(req.EndsAt != nil && !req.EndsAt.Equal(existing.EndsAt))

	if needsCheckBR_E5 {
		activeRegCount, err := s.eventRepo.CountActiveRegistrations(ctx, id)
		if err == nil && activeRegCount > 0 {
			return helper.ConflictCode("HAS_ACTIVE_REGISTRATIONS", "event dengan registrasi aktif tidak boleh diubah kapasitas atau waktunya")
		}
	}

	if req.Capacity != nil {
		fields["capacity"] = *req.Capacity
	}
	if req.StartsAt != nil {
		fields["starts_at"] = *req.StartsAt
	}
	if req.EndsAt != nil {
		fields["ends_at"] = *req.EndsAt
	}

	updated, err := s.eventRepo.Patch(ctx, id, fields)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, "event berhasil diperbarui", updated)
}

// #22 DELETE /events/:id
func (s *EventService) DeleteEvent(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	existing, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	viewerID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	viewerRole, _ := helper.CurrentUserRole(c)

	if !CanManageEvent(viewerID, viewerRole, existing.OrganizerID, "event:delete:any") {
		return helper.Forbidden("tidak memiliki akses untuk menghapus event ini")
	}

	// BR-E6: Hanya event draft yang boleh dihapus
	if err := ValidateEventDeletion(existing.Status); err != nil {
		return err
	}

	err = s.eventRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.NoContent(c)
}

// #23 PATCH /events/:id/status
func (s *EventService) UpdateEventStatus(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	var req model.UpdateEventStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	existing, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	viewerID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	viewerRole, _ := helper.CurrentUserRole(c)

	if !CanManageEvent(viewerID, viewerRole, existing.OrganizerID, "event:update:any") {
		return helper.Forbidden("tidak memiliki akses untuk mengubah status event ini")
	}

	// Validasi transisi state machine via pure rules
	if err := ValidateEventStatusTransition(existing.Status, req.Status, time.Now(), existing.EndsAt); err != nil {
		return err
	}

	// BR-E7: Event dengan registrasi aktif tidak boleh dibatalkan (kecuali MVP rule / spesifikasi khusus)
	if req.Status == "cancelled" {
		activeRegCount, err := s.eventRepo.CountActiveRegistrations(ctx, id)
		if err == nil && activeRegCount > 0 {
			return helper.ConflictCode("HAS_ACTIVE_REGISTRATIONS", "event dengan registrasi aktif tidak boleh dibatalkan")
		}
	}

	updated, err := s.eventRepo.UpdateStatus(ctx, id, req.Status)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, "status event berhasil diperbarui", updated)
}

// #24 GET /events/:id/ticket-types
func (s *EventService) GetTicketTypes(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	viewerRole := ""
	if r, err := helper.CurrentUserRole(c); err == nil {
		viewerRole = r
	}
	viewerID := 0
	if idVal, err := helper.CurrentUserID(c); err == nil {
		viewerID = idVal
	}

	if !CanReadEvent(viewerRole, event.Status, viewerID, event.OrganizerID) {
		return helper.NotFound("event tidak ditemukan")
	}

	ticketTypes, err := s.ticketTypeRepo.FindByEventID(ctx, id)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, "daftar jenis tiket ditemukan", ticketTypes)
}

// #25 POST /events/:id/ticket-types
func (s *EventService) CreateTicketType(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	var req model.CreateTicketTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	event, err := s.eventRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	viewerID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	viewerRole, _ := helper.CurrentUserRole(c)

	if !CanManageEvent(viewerID, viewerRole, event.OrganizerID, "event:update:any") {
		return helper.Forbidden("tidak memiliki akses untuk mengelola tiket event ini")
	}

	// BR-T1: Maksimal 10 jenis tiket per event
	existingTypes, err := s.ticketTypeRepo.FindByEventID(ctx, id)
	if err != nil {
		return helper.Internal(err)
	}
	if len(existingTypes) >= 10 {
		return helper.ConflictCode("MAX_TICKET_TYPES", "maksimal 10 jenis tiket per event")
	}

	// BR-E3: Total kuota jenis tiket <= kapasitas event
	currentSumQuota, err := s.ticketTypeRepo.SumQuotaByEventID(ctx, id)
	if err != nil {
		return helper.Internal(err)
	}
	if err := ValidateTicketTypeQuota(event.Capacity, currentSumQuota, req.Quota); err != nil {
		return err
	}

	created, err := s.ticketTypeRepo.Create(ctx, id, &req)
	if err != nil {
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			return helper.Conflict("nama jenis tiket sudah digunakan pada event ini")
		}
		return helper.Internal(err)
	}

	location := fmt.Sprintf("/api/v1/ticket-types/%d", created.ID)
	return helper.Created(c, location, "jenis tiket berhasil dibuat", created)
}

// #26 PUT /ticket-types/:id
func (s *EventService) UpdateTicketType(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	var req model.PutTicketTypeRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	ticketType, err := s.ticketTypeRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("jenis tiket tidak ditemukan")
		}
		return helper.Internal(err)
	}

	event, err := s.eventRepo.FindByID(ctx, ticketType.EventID)
	if err != nil {
		return helper.NotFound("event induk tidak ditemukan")
	}

	viewerID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	viewerRole, _ := helper.CurrentUserRole(c)

	if !CanManageEvent(viewerID, viewerRole, event.OrganizerID, "event:update:any") {
		return helper.Forbidden("tidak memiliki akses untuk mengubah jenis tiket ini")
	}

	// Cek kuota baru vs kapasitas (kurangi kuota lama yang sedang di-update)
	currentSumQuota, err := s.ticketTypeRepo.SumQuotaByEventID(ctx, event.ID)
	if err != nil {
		return helper.Internal(err)
	}
	otherQuotaSum := currentSumQuota - ticketType.Quota
	if err := ValidateTicketTypeQuota(event.Capacity, otherQuotaSum, req.Quota); err != nil {
		return err
	}

	// Kuota tidak boleh lebih kecil dari yang sudah terjual
	if req.Quota < ticketType.Sold {
		return helper.Validation("validasi gagal", map[string]string{"quota": "kuota tidak boleh lebih kecil dari jumlah tiket yang sudah terjual"})
	}

	updated, err := s.ticketTypeRepo.Update(ctx, id, &req)
	if err != nil {
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			return helper.Conflict("nama jenis tiket sudah digunakan pada event ini")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, "jenis tiket berhasil diperbarui", updated)
}

// #27 DELETE /ticket-types/:id
func (s *EventService) DeleteTicketType(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	ticketType, err := s.ticketTypeRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("jenis tiket tidak ditemukan")
		}
		return helper.Internal(err)
	}

	event, err := s.eventRepo.FindByID(ctx, ticketType.EventID)
	if err != nil {
		return helper.NotFound("event induk tidak ditemukan")
	}

	viewerID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	viewerRole, _ := helper.CurrentUserRole(c)

	if !CanManageEvent(viewerID, viewerRole, event.OrganizerID, "event:update:any") {
		return helper.Forbidden("tidak memiliki akses untuk menghapus jenis tiket ini")
	}

	// BR-T4: Jenis tiket yang sudah ada pendaftaran tidak boleh dihapus
	regCount, err := s.ticketTypeRepo.CountRegistrations(ctx, id)
	if err == nil && regCount > 0 {
		return helper.ConflictCode("HAS_REGISTRATIONS", "jenis tiket yang sudah memiliki pendaftaran aktif tidak dapat dihapus")
	}

	err = s.ticketTypeRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("jenis tiket tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.NoContent(c)
}
