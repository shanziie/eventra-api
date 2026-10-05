package helper

import "net/http"

// AppError adalah error standar aplikasi yang membawa informasi HTTP.
// cause sengaja tidak diekspor agar detail teknis tidak bocor ke layer lain;
// Error() sudah menyertakan pesan cause sehingga log tetap informatif.
type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	cause   error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.cause
}

// --- konstruktor AppError ---

func BadRequest(message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: message}
}

func Unauthorized(message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: message}
}

func NotFound(message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "CONFLICT", Message: message}
}

// ConflictCode membuat error 409 dengan kode domain khusus (mis. QUOTA_FULL).
func ConflictCode(code, message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: code, Message: message}
}

// Validation membuat error 422 dengan map pelanggaran field.
func Validation(message string, fields map[string]string) *AppError {
	return &AppError{
		Status:  http.StatusUnprocessableEntity,
		Code:    "VALIDATION_ERROR",
		Message: message,
		Fields:  fields,
	}
}

func NotAcceptable(message string) *AppError {
	return &AppError{Status: http.StatusNotAcceptable, Code: "NOT_ACCEPTABLE", Message: message}
}

func UnsupportedMediaType(message string) *AppError {
	return &AppError{
		Status:  http.StatusUnsupportedMediaType,
		Code:    "UNSUPPORTED_MEDIA_TYPE",
		Message: message,
	}
}

func TooManyRequests(message string) *AppError {
	return &AppError{
		Status:  http.StatusTooManyRequests,
		Code:    "TOO_MANY_REQUESTS",
		Message: message,
	}
}

func ServiceUnavailable(message string) *AppError {
	return &AppError{
		Status:  http.StatusServiceUnavailable,
		Code:    "SERVICE_UNAVAILABLE",
		Message: message,
	}
}

// Internal membuat error 500 dan menyimpan cause di field tidak diekspor.
// Pesan yang tampil ke client selalu generik; detail hanya di log via Error().
func Internal(cause error) *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_ERROR",
		Message: "terjadi kesalahan internal",
		cause:   cause,
	}
}
