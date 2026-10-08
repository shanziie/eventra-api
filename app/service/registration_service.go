package service

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
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

type RegistrationService struct {
	regRepo        repository.RegistrationRepository
	eventRepo      repository.EventRepository
	ticketTypeRepo repository.TicketTypeRepository
	paymentRepo    repository.PaymentRepository
}

func NewRegistrationService(
	regRepo repository.RegistrationRepository,
	eventRepo repository.EventRepository,
	ticketTypeRepo repository.TicketTypeRepository,
	paymentRepo repository.PaymentRepository,
) *RegistrationService {
	return &RegistrationService{
		regRepo:        regRepo,
		eventRepo:      eventRepo,
		ticketTypeRepo: ticketTypeRepo,
		paymentRepo:    paymentRepo,
	}
}

// #28 POST /events/:id/registrations
func (s *RegistrationService) CreateRegistration(c *fiber.Ctx) error {
	eventID, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	var req model.CreateRegistrationRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	userID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// 1. Ambil event
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// BR-R1: Event harus published dan belum lewat registration_deadline
	if err := ValidateRegistrationCreation(event.Status, event.RegistrationDeadline, time.Now()); err != nil {
		return err
	}

	// 2. Ambil jenis tiket
	ticketType, err := s.ticketTypeRepo.FindByID(ctx, req.TicketTypeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("jenis tiket tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// BR-R2: Jenis tiket harus milik event yang didaftar
	if err := ValidateTicketTypeBelongsToEvent(ticketType.EventID, eventID); err != nil {
		return err
	}

	// BR-R3: Satu user hanya boleh punya satu registrasi aktif (non-cancelled) per event
	// Kita bisa cek via find by user id / filter event_id atau constraint unique index
	userRegs, _, _, err := s.regRepo.FindByUserID(ctx, userID, 100, nil, nil, "")
	if err == nil {
		for _, r := range userRegs {
			if r.EventID == eventID && r.Status != "cancelled" {
				return helper.ConflictCode("ALREADY_REGISTERED", "anda sudah memiliki pendaftaran aktif pada event ini")
			}
		}
	}

	// 3. Buat registrasi (transaksional dengan locking FOR UPDATE)
	created, err := s.regRepo.Create(ctx, userID, eventID, req.TicketTypeID, int64(ticketType.Price))
	if err != nil {
		if errors.Is(err, repository.ErrQuotaFull) {
			return helper.ConflictCode("QUOTA_FULL", "kuota jenis tiket sudah habis")
		}
		return helper.Internal(err)
	}

	location := fmt.Sprintf("/api/v1/registrations/%d", created.ID)
	return helper.Created(c, location, "pendaftaran berhasil dibuat", created)
}

// #29 GET /events/:id/registrations (termasuk export CSV jika Accept: text/csv)
func (s *RegistrationService) GetRegistrationsByEvent(c *fiber.Ctx) error {
	eventID, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("event tidak ditemukan")
		}
		return helper.Internal(err)
	}

	userID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	role, _ := helper.CurrentUserRole(c)

	if !CanAccessRegistration(userID, role, 0, event.OrganizerID, "registration:read:any") {
		return helper.Forbidden("tidak memiliki akses melihat daftar registrasi event ini")
	}

	status := c.Query("status")
	acceptHeader := c.Get("Accept")

	if strings.Contains(acceptHeader, "text/csv") {
		regs, err := s.regRepo.FindAllForExport(ctx, eventID, status)
		if err != nil {
			return helper.Internal(err)
		}

		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="registrations_event_%d.csv"`, eventID))

		var csvBuilder strings.Builder
		writer := csv.NewWriter(&csvBuilder)
		_ = writer.Write([]string{"ID", "Username", "Email", "Full Name", "Event Title", "Ticket Name", "Status", "Price", "Ticket Code", "Created At"})

		for _, r := range regs {
			username := ""
			if r.Username != nil {
				username = *r.Username
			}
			email := ""
			if r.Email != nil {
				email = *r.Email
			}
			fullName := ""
			if r.FullName != nil {
				fullName = *r.FullName
			}
			eventTitle := ""
			if r.EventTitle != nil {
				eventTitle = *r.EventTitle
			}
			ticketName := ""
			if r.TicketName != nil {
				ticketName = *r.TicketName
			}
			ticketCode := ""
			if r.TicketCode != nil {
				ticketCode = *r.TicketCode
			}

			_ = writer.Write([]string{
				strconv.Itoa(r.ID),
				username,
				email,
				fullName,
				eventTitle,
				ticketName,
				r.Status,
				strconv.FormatInt(r.PriceAtPurchase, 10),
				ticketCode,
				r.CreatedAt.Format(time.RFC3339),
			})
		}
		writer.Flush()
		return c.SendString(csvBuilder.String())
	}

	// Normal Cursor Pagination JSON response
	limit := 10
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v >= 1 && v <= 100 {
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

	regs, nextCursor, hasMore, err := s.regRepo.FindByEventID(ctx, eventID, limit, cursorTime, cursorID, status)
	if err != nil {
		return helper.Internal(err)
	}

	res := model.RegistrationCursorResponse{
		Data: regs,
	}
	res.Pagination.Limit = limit
	res.Pagination.NextCursor = nextCursor
	res.Pagination.HasMore = hasMore

	return helper.Success(c, "daftar registrasi ditemukan", res)
}

// #30 GET /registrations/:id
func (s *RegistrationService) GetRegistrationByID(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	reg, err := s.regRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pendaftaran tidak ditemukan")
		}
		return helper.Internal(err)
	}

	event, err := s.eventRepo.FindByID(ctx, reg.EventID)
	if err != nil {
		return helper.NotFound("event tidak ditemukan")
	}

	userID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	role, _ := helper.CurrentUserRole(c)

	if !CanAccessRegistration(userID, role, reg.UserID, event.OrganizerID, "registration:read:any") {
		return helper.Forbidden("tidak memiliki akses melihat pendaftaran ini")
	}

	payment, _ := s.paymentRepo.FindByRegistrationID(ctx, id)

	return helper.Success(c, "detail pendaftaran ditemukan", fiber.Map{
		"registration": reg,
		"payment":      payment,
	})
}

// #31 POST /registrations/:id/payment
func (s *RegistrationService) SubmitPayment(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	var req model.SubmitPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	reg, err := s.regRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pendaftaran tidak ditemukan")
		}
		return helper.Internal(err)
	}

	userID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}

	if reg.UserID != userID {
		return helper.Forbidden("hanya pemilik pendaftaran yang dapat mengirimkan pembayaran")
	}

	payment, err := s.paymentRepo.Submit(ctx, id, req.Method, req.ReferenceNumber)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("data pembayaran tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, "bukti pembayaran berhasil dikirim", payment)
}

// #32 PATCH /registrations/:id/payment (Organizer verifikasi pembayaran)
func (s *RegistrationService) VerifyPayment(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	var req model.VerifyPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	reg, err := s.regRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pendaftaran tidak ditemukan")
		}
		return helper.Internal(err)
	}

	event, err := s.eventRepo.FindByID(ctx, reg.EventID)
	if err != nil {
		return helper.NotFound("event tidak ditemukan")
	}

	userID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	role, _ := helper.CurrentUserRole(c)

	if !CanAccessRegistration(userID, role, 0, event.OrganizerID, "registration:verify:any") {
		return helper.Forbidden("tidak memiliki akses memverifikasi pembayaran ini")
	}

	payment, err := s.paymentRepo.Verify(ctx, id, req.Status, req.Note, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pendaftaran tidak ditemukan")
		}
		return helper.Internal(err)
	}

	// Jika verified, update status registrasi menjadi 'confirmed' dan generate ticket_code unik
	var newRegStatus = "pending"
	var ticketCode *string
	if req.Status == "verified" {
		newRegStatus = "confirmed"
		code := generateTicketCode(reg.EventID, reg.ID)
		ticketCode = &code
	} else if req.Status == "rejected" {
		newRegStatus = "pending"
	}

	updatedReg, err := s.regRepo.UpdateStatus(ctx, id, newRegStatus, ticketCode)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, "status pembayaran berhasil diperbarui", fiber.Map{
		"registration": updatedReg,
		"payment":      payment,
	})
}

// #33 DELETE /registrations/:id (Batalkan pendaftaran, maks H-24)
func (s *RegistrationService) CancelRegistration(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	reg, err := s.regRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pendaftaran tidak ditemukan")
		}
		return helper.Internal(err)
	}

	event, err := s.eventRepo.FindByID(ctx, reg.EventID)
	if err != nil {
		return helper.NotFound("event tidak ditemukan")
	}

	userID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	role, _ := helper.CurrentUserRole(c)

	if !CanAccessRegistration(userID, role, reg.UserID, event.OrganizerID, "registration:delete:any") {
		return helper.Forbidden("tidak memiliki akses membatalkan pendaftaran ini")
	}

	// BR-R4: Pembatalan maksimal H-24 sebelum event mulai
	if err := ValidateCancellationWindow(event.StartsAt, time.Now()); err != nil {
		return err
	}

	cancelled, err := s.regRepo.Cancel(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pendaftaran tidak ditemukan")
		}
		if errors.Is(err, repository.ErrAlreadyCancelled) {
			return helper.Conflict("pendaftaran sudah dibatalkan sebelumnya")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, "pendaftaran berhasil dibatalkan", cancelled)
}

// #34 POST /registrations/:id/check-in
func (s *RegistrationService) CheckInRegistration(c *fiber.Ctx) error {
	id, appErr := helper.ParamID(c)
	if appErr != nil {
		return appErr
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	reg, err := s.regRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pendaftaran tidak ditemukan")
		}
		return helper.Internal(err)
	}

	event, err := s.eventRepo.FindByID(ctx, reg.EventID)
	if err != nil {
		return helper.NotFound("event tidak ditemukan")
	}

	userID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	role, _ := helper.CurrentUserRole(c)

	if !CanAccessRegistration(userID, role, 0, event.OrganizerID, "registration:checkin:any") {
		return helper.Forbidden("tidak memiliki akses melakukan check-in untuk event ini")
	}

	if reg.Status != "confirmed" {
		if reg.Status == "checked_in" {
			return helper.ConflictCode("ALREADY_CHECKED_IN", "tiket sudah pernah digunakan untuk check-in")
		}
		return helper.ConflictCode("NOT_CONFIRMED", "pendaftaran belum dikonfirmasi/lunas")
	}

	// Validasi jendela check-in
	if err := ValidateCheckInWindow(event.StartsAt, event.EndsAt, time.Now()); err != nil {
		return err
	}

	checkedIn, err := s.regRepo.CheckIn(ctx, id)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, "check-in berhasil", checkedIn)
}

// #35 GET /users/:id/registrations
func (s *RegistrationService) GetRegistrationsByUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	targetUserID, err := strconv.Atoi(idParam)
	if err != nil {
		return helper.BadRequest("id pengguna tidak valid")
	}

	userID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}
	role, _ := helper.CurrentUserRole(c)

	if !CanAccessUser(userID, role, targetUserID, "user:read:any") {
		return helper.Forbidden("tidak memiliki akses melihat registrasi pengguna ini")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	limit := 10
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v >= 1 && v <= 100 {
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

	status := c.Query("status")

	regs, nextCursor, hasMore, err := s.regRepo.FindByUserID(ctx, targetUserID, limit, cursorTime, cursorID, status)
	if err != nil {
		return helper.Internal(err)
	}

	res := model.RegistrationCursorResponse{
		Data: regs,
	}
	res.Pagination.Limit = limit
	res.Pagination.NextCursor = nextCursor
	res.Pagination.HasMore = hasMore

	return helper.Success(c, "daftar registrasi pengguna ditemukan", res)
}

func generateTicketCode(eventID, regID int) string {
	bytes := make([]byte, 4)
	_, _ = rand.Read(bytes)
	return fmt.Sprintf("EVT-%d-REG-%d-%s", eventID, regID, hex.EncodeToString(bytes))
}
