package helper

import (
	"context"
	"strconv"
	"time"

	"eventra-api/app/model"

	"github.com/gofiber/fiber/v2"
)

// RequestContext mengembalikan context dengan timeout 5 detik untuk setiap operasi DB.
// Caller wajib memanggil cancel() via defer.
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Context(), 5*time.Second)
}

// ParamID membaca parameter ":id" dari URL dan memvalidasi bahwa nilainya positif.
func ParamID(c *fiber.Ctx) (int, *AppError) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, BadRequest("id tidak sah")
	}
	return id, nil
}

// OffsetQuery berisi parameter pagination offset yang sudah divalidasi.
type OffsetQuery struct {
	Page   int
	Limit  int
	Search string
	Sort   string
	Order  string
}

// ReadOffsetQuery membaca parameter pagination dari query string.
// sortWhitelist memetakan nama query ke nama kolom SQL yang diizinkan,
// untuk mencegah SQL injection via ORDER BY.
func ReadOffsetQuery(c *fiber.Ctx, sortWhitelist map[string]string) (OffsetQuery, *AppError) {
	q := OffsetQuery{
		Page:   1,
		Limit:  10,
		Search: c.Query("search"),
		Order:  "asc",
	}

	if p := c.Query("page"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil || v < 1 {
			return q, BadRequest("page harus angka positif")
		}
		q.Page = v
	}

	if l := c.Query("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err != nil || v < 1 || v > 100 {
			return q, BadRequest("limit harus antara 1 dan 100")
		}
		q.Limit = v
	}

	if s := c.Query("sort"); s != "" {
		col, ok := sortWhitelist[s]
		if !ok {
			return q, BadRequest("sort tidak diizinkan: " + s)
		}
		q.Sort = col
	}

	if o := c.Query("order"); o != "" {
		if o != "asc" && o != "desc" {
			return q, BadRequest("order harus asc atau desc")
		}
		q.Order = o
	}

	return q, nil
}

// CalculateMeta menghitung informasi pagination offset (page, limit, total, total_pages).
func CalculateMeta(page, limit, total int) model.Meta {
	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}
	return model.Meta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
