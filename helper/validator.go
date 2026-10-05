package helper

import (
	"errors"
	"reflect"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// validate adalah singleton; dibuat sekali saat startup untuk efisiensi.
var validate *validator.Validate

func init() {
	validate = validator.New()

	// Pakai nama field JSON sebagai nama field di pesan error,
	// bukan nama struct field Go (misalnya "full_name" bukan "FullName").
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	validate.RegisterValidation("nospace", validateNoSpace)
	validate.RegisterValidation("username", validateUsername)
	validate.RegisterValidation("strongpassword", validateStrongPassword)
}

// validateNoSpace memastikan tidak ada spasi atau karakter whitespace.
func validateNoSpace(fl validator.FieldLevel) bool {
	return !strings.ContainsAny(fl.Field().String(), " \t\r\n")
}

// validateUsername memastikan hanya huruf, angka, underscore, dan dash.
func validateUsername(fl validator.FieldLevel) bool {
	for _, r := range fl.Field().String() {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

// validateStrongPassword membungkus IsStrongPassword untuk dipakai sebagai tag validator.
func validateStrongPassword(fl validator.FieldLevel) bool {
	return IsStrongPassword(fl.Field().String())
}

// IsStrongPassword memeriksa apakah password cukup kuat:
// minimal 8 karakter, mengandung huruf besar, huruf kecil, angka, dan simbol.
// Fungsi ini diekspor agar bisa diuji langsung tanpa Fiber.
func IsStrongPassword(s string) bool {
	if len(s) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSymbol = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSymbol
}

// ValidateStruct memvalidasi struct dan mengumpulkan SEMUA pelanggaran sekaligus.
// Mengembalikan nil bila tidak ada pelanggaran.
func ValidateStruct(s any) *AppError {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		// Bukan error validasi biasa (mis. invalid type) → 500
		return Internal(err)
	}

	fields := make(map[string]string, len(ve))
	for _, fe := range ve {
		fields[fe.Field()] = messageFor(fe)
	}
	return Validation("validasi gagal", fields)
}

// messageFor mengembalikan pesan validasi Bahasa Indonesia untuk satu pelanggaran.
// default-nya tidak pernah kosong sesuai AGENTS.md.
func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		return "minimal " + fe.Param() + " karakter"
	case "max":
		return "maksimal " + fe.Param() + " karakter"
	case "len":
		return "harus tepat " + fe.Param() + " karakter"
	case "oneof":
		return "nilai harus salah satu dari: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "nospace":
		return "tidak boleh mengandung spasi"
	case "username":
		return "hanya boleh huruf, angka, underscore, dan dash"
	case "strongpassword":
		return "password harus memuat huruf besar, huruf kecil, angka, dan simbol (minimal 8 karakter)"
	case "url":
		return "format URL tidak valid"
	default:
		return "nilai tidak valid"
	}
}
