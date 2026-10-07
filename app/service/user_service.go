package service

import (
	"errors"
	"strconv"
	"strings"

	"eventra-api/app/model"
	"eventra-api/app/repository"
	"eventra-api/helper"

	"github.com/gofiber/fiber/v2"
)

// UserService mengelola logika bisnis dan otorisasi pengguna.
type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// GetUsers mengambil daftar pengguna dengan pagination offset dan filter (memerlukan user:list).
func (s *UserService) GetUsers(c *fiber.Ctx) error {
	sortWhitelist := map[string]string{
		"created_at": "created_at",
		"username":   "username",
		"email":      "email",
		"full_name":  "full_name",
	}
	q, appErr := helper.ReadOffsetQuery(c, sortWhitelist)
	if appErr != nil {
		return appErr
	}
	offset := (q.Page - 1) * q.Limit

	role := c.Query("role")

	var isActive *bool
	if activeStr := c.Query("is_active"); activeStr != "" {
		if val, err := strconv.ParseBool(activeStr); err == nil {
			isActive = &val
		}
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	users, total, err := s.userRepo.FindAll(ctx, q.Limit, offset, q.Search, role, isActive, q.Sort, q.Order)
	if err != nil {
		return helper.Internal(err)
	}

	authUsers := make([]model.AuthUser, len(users))
	for i, u := range users {
		authUsers[i] = model.AuthUser{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			FullName:  u.FullName,
			Role:      u.Role,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt,
		}
	}

	meta := helper.CalculateMeta(q.Page, q.Limit, total)
	return helper.SuccessList(c, "daftar pengguna ditemukan", authUsers, meta)
}

// GetUserByID mengambil detail profil pengguna (ownership atau user:read:any).
func (s *UserService) GetUserByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	targetID, err := strconv.Atoi(idParam)
	if err != nil {
		return helper.BadRequest("id pengguna tidak valid")
	}

	currentUserID := c.Locals("user_id").(int)
	currentRole := c.Locals("role").(string)

	if !CanAccessUser(currentUserID, currentRole, targetID, "user:read:any") {
		return helper.Forbidden("anda tidak memiliki izin untuk melihat data pengguna ini")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, err := s.userRepo.FindByID(ctx, targetID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pengguna tidak ditemukan")
		}
		return helper.Internal(err)
	}

	authUser := model.AuthUser{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}

	return helper.Success(c, "profil pengguna ditemukan", authUser)
}

// PutUser memperbarui penuh profil pengguna (ownership atau user:update:any).
func (s *UserService) PutUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	targetID, err := strconv.Atoi(idParam)
	if err != nil {
		return helper.BadRequest("id pengguna tidak valid")
	}

	currentUserID := c.Locals("user_id").(int)
	currentRole := c.Locals("role").(string)

	if !CanAccessUser(currentUserID, currentRole, targetID, "user:update:any") {
		return helper.Forbidden("anda tidak memiliki izin untuk mengubah data pengguna ini")
	}

	var req model.PutUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	// BR-U2: is_active hanya boleh diubah oleh pemegang user:update:any (bukan pemilik yang bukan admin)
	if currentUserID == targetID && !helper.Can(currentRole, "user:update:any") {
		// Jika user biasa mengubah datanya sendiri, is_active harus tetap sesuai aslinya
		ctxCheck, cancelCheck := helper.RequestContext(c)
		existing, err := s.userRepo.FindByID(ctxCheck, targetID)
		cancelCheck()
		if err == nil && req.IsActive != nil && *req.IsActive != existing.IsActive {
			return helper.Forbidden("anda tidak berhak mengubah status aktif akun")
		}
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	updated, err := s.userRepo.Update(ctx, targetID, &req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pengguna tidak ditemukan")
		}
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			cons := strings.ToLower(dupErr.Constraint)
			if strings.Contains(cons, "username") {
				return helper.Conflict("username sudah digunakan")
			}
			if strings.Contains(cons, "email") {
				return helper.Conflict("email sudah digunakan")
			}
			return helper.Conflict("username atau email sudah terdaftar")
		}
		return helper.Internal(err)
	}

	authUser := model.AuthUser{
		ID:        updated.ID,
		Username:  updated.Username,
		Email:     updated.Email,
		FullName:  updated.FullName,
		Role:      updated.Role,
		IsActive:  updated.IsActive,
		CreatedAt: updated.CreatedAt,
	}

	return helper.Success(c, "profil pengguna berhasil diperbarui", authUser)
}

// PatchUser memperbarui sebagian profil pengguna (ownership atau user:update:any).
func (s *UserService) PatchUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	targetID, err := strconv.Atoi(idParam)
	if err != nil {
		return helper.BadRequest("id pengguna tidak valid")
	}

	currentUserID := c.Locals("user_id").(int)
	currentRole := c.Locals("role").(string)

	if !CanAccessUser(currentUserID, currentRole, targetID, "user:update:any") {
		return helper.Forbidden("anda tidak memiliki izin untuk mengubah data pengguna ini")
	}

	var req model.PatchUserRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	if req.Username == nil && req.Email == nil && req.FullName == nil && req.IsActive == nil {
		return helper.BadRequest("body request tidak boleh kosong")
	}

	// BR-U2: is_active hanya boleh diubah oleh pemegang user:update:any
	if req.IsActive != nil && currentUserID == targetID && !helper.Can(currentRole, "user:update:any") {
		return helper.Forbidden("anda tidak berhak mengubah status aktif akun")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	updated, err := s.userRepo.Patch(ctx, targetID, req.Username, req.Email, req.FullName, req.IsActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pengguna tidak ditemukan")
		}
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			cons := strings.ToLower(dupErr.Constraint)
			if strings.Contains(cons, "username") {
				return helper.Conflict("username sudah digunakan")
			}
			if strings.Contains(cons, "email") {
				return helper.Conflict("email sudah digunakan")
			}
			return helper.Conflict("username atau email sudah terdaftar")
		}
		return helper.Internal(err)
	}

	authUser := model.AuthUser{
		ID:        updated.ID,
		Username:  updated.Username,
		Email:     updated.Email,
		FullName:  updated.FullName,
		Role:      updated.Role,
		IsActive:  updated.IsActive,
		CreatedAt: updated.CreatedAt,
	}

	return helper.Success(c, "profil pengguna berhasil diperbarui", authUser)
}

// DeleteUser menghapus pengguna (BR-U4: tidak boleh hapus diri sendiri; BR-U5: ada event/registrasi tidak bisa dihapus).
func (s *UserService) DeleteUser(c *fiber.Ctx) error {
	idParam := c.Params("id")
	targetID, err := strconv.Atoi(idParam)
	if err != nil {
		return helper.BadRequest("id pengguna tidak valid")
	}

	currentUserID := c.Locals("user_id").(int)

	// BR-U4: Tidak boleh menghapus akun sendiri
	if currentUserID == targetID {
		return helper.Forbidden("tidak boleh menghapus akun sendiri")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// BR-U5: User yang masih jadi organizer event atau punya registrasi tidak bisa dihapus
	eventCount, err := s.userRepo.CountEventsByOrganizer(ctx, targetID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return helper.Internal(err)
	}
	regCount, err := s.userRepo.CountRegistrationsByUser(ctx, targetID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return helper.Internal(err)
	}

	if eventCount > 0 || regCount > 0 {
		return helper.ConflictCode("HAS_DEPENDENCIES", "pengguna masih memiliki event atau registrasi, nonaktifkan saja")
	}

	err = s.userRepo.Delete(ctx, targetID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pengguna tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.NoContent(c)
}

// AssignRole mengubah role pengguna (BR-U4: tidak boleh mengubah role diri sendiri; role harus sah).
func (s *UserService) AssignRole(c *fiber.Ctx) error {
	idParam := c.Params("id")
	targetID, err := strconv.Atoi(idParam)
	if err != nil {
		return helper.BadRequest("id pengguna tidak valid")
	}

	currentUserID := c.Locals("user_id").(int)

	var req model.AssignRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	// Sesuai ValidateAssignRole (BR-U4)
	if err := ValidateAssignRole(currentUserID, targetID, req.Role); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	updated, err := s.userRepo.UpdateRole(ctx, targetID, req.Role)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pengguna tidak ditemukan")
		}
		return helper.Internal(err)
	}

	authUser := model.AuthUser{
		ID:        updated.ID,
		Username:  updated.Username,
		Email:     updated.Email,
		FullName:  updated.FullName,
		Role:      updated.Role,
		IsActive:  updated.IsActive,
		CreatedAt: updated.CreatedAt,
	}

	return helper.Success(c, "role pengguna berhasil diubah", authUser)
}
