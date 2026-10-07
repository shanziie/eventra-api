package service

import (
	"errors"
	"fmt"
	"strconv"

	"eventra-api/app/model"
	"eventra-api/app/repository"
	"eventra-api/helper"

	"github.com/gofiber/fiber/v2"
)

// CategoryService mengelola logika bisnis kategori event.
type CategoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) *CategoryService {
	return &CategoryService{categoryRepo: categoryRepo}
}

// GetCategories mengambil daftar seluruh kategori event.
func (s *CategoryService) GetCategories(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	categories, err := s.categoryRepo.FindAll(ctx)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, "daftar kategori ditemukan", categories)
}

// CreateCategory membuat kategori event baru (memerlukan permission category:create).
func (s *CategoryService) CreateCategory(c *fiber.Ctx) error {
	var req model.CategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	created, err := s.categoryRepo.Create(ctx, &req)
	if err != nil {
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			return helper.Conflict("nama kategori sudah digunakan")
		}
		return helper.Internal(err)
	}

	location := fmt.Sprintf("/api/v1/categories/%d", created.ID)
	return helper.Created(c, location, "kategori berhasil dibuat", created)
}

// UpdateCategory mengubah data kategori event (memerlukan permission category:update).
func (s *CategoryService) UpdateCategory(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return helper.BadRequest("id kategori tidak valid")
	}

	var req model.CategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	updated, err := s.categoryRepo.Update(ctx, id, &req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("kategori tidak ditemukan")
		}
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			return helper.Conflict("nama kategori sudah digunakan")
		}
		return helper.Internal(err)
	}

	return helper.Success(c, "kategori berhasil diperbarui", updated)
}

// DeleteCategory menghapus kategori event (BR-C2: tidak boleh dihapus jika masih dipakai event).
func (s *CategoryService) DeleteCategory(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		return helper.BadRequest("id kategori tidak valid")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// BR-C2: Kategori yang masih dipakai event tidak bisa dihapus
	count, err := s.categoryRepo.CountEvents(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("kategori tidak ditemukan")
		}
		return helper.Internal(err)
	}
	if count > 0 {
		return helper.ConflictCode("HAS_DEPENDENCIES", "kategori masih digunakan oleh event")
	}

	err = s.categoryRepo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("kategori tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.NoContent(c)
}
