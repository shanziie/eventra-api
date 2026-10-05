package helper_test

import (
	"testing"

	"eventra-api/helper"
)

// --- test ValidateStruct ---

// reqDua adalah struct uji dengan dua field bervalidasi.
type reqDua struct {
	Username string `json:"username" validate:"required,min=3"`
	Email    string `json:"email"    validate:"required,email"`
}

// reqLengkap menguji beberapa tag sekaligus.
type reqLengkap struct {
	Name     string `json:"name"     validate:"required,min=2,max=50"`
	Password string `json:"password" validate:"required,strongpassword"`
	Code     string `json:"code"     validate:"required,nospace"`
	Handle   string `json:"handle"   validate:"required,username"`
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name       string
		input      any
		wantErr    bool
		wantFields []string // nama field JSON yang diharapkan muncul di Fields
	}{
		{
			name:    "valid - tidak ada pelanggaran",
			input:   reqDua{Username: "alice", Email: "alice@example.com"},
			wantErr: false,
		},
		{
			name:       "semua field kosong - semua pelanggaran dilaporkan sekaligus",
			input:      reqDua{},
			wantErr:    true,
			wantFields: []string{"username", "email"},
		},
		{
			name:       "email tidak valid - field memakai nama JSON bukan struct",
			input:      reqDua{Username: "alice", Email: "bukan-email"},
			wantErr:    true,
			wantFields: []string{"email"},
		},
		{
			name:       "username terlalu pendek",
			input:      reqDua{Username: "ab", Email: "alice@example.com"},
			wantErr:    true,
			wantFields: []string{"username"},
		},
		{
			name: "nospace - ada spasi",
			input: reqLengkap{
				Name: "ok", Password: "Passw0rd!", Code: "ada spasi", Handle: "valid",
			},
			wantErr:    true,
			wantFields: []string{"code"},
		},
		{
			name: "username - karakter tidak valid",
			input: reqLengkap{
				Name: "ok", Password: "Passw0rd!", Code: "nospace", Handle: "invalid karakter!",
			},
			wantErr:    true,
			wantFields: []string{"handle"},
		},
		{
			name: "strongpassword - password lemah",
			input: reqLengkap{
				Name: "ok", Password: "lemah", Code: "nospace", Handle: "valid",
			},
			wantErr:    true,
			wantFields: []string{"password"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			appErr := helper.ValidateStruct(tc.input)

			if tc.wantErr && appErr == nil {
				t.Fatal("diharapkan error, dapat nil")
			}
			if !tc.wantErr && appErr != nil {
				t.Fatalf("diharapkan tidak ada error, dapat: %v", appErr)
			}
			if appErr == nil {
				return
			}

			// Periksa setiap field yang diharapkan ada dan pesannya tidak kosong.
			for _, field := range tc.wantFields {
				msg, ok := appErr.Fields[field]
				if !ok {
					t.Errorf("field %q tidak ditemukan di Fields %v", field, appErr.Fields)
					continue
				}
				if msg == "" {
					t.Errorf("pesan untuk field %q kosong", field)
				}
			}
		})
	}
}

// --- test IsStrongPassword ---

func TestIsStrongPassword(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"kuat - semua kriteria terpenuhi", "Passw0rd!", true},
		{"kuat - lebih kompleks", "G0!Super#Secure2", true},
		{"lemah - kurang dari 8 karakter", "Ab1!", false},
		{"lemah - tidak ada huruf besar", "passw0rd!", false},
		{"lemah - tidak ada huruf kecil", "PASSW0RD!", false},
		{"lemah - tidak ada angka", "Password!", false},
		{"lemah - tidak ada simbol", "Passw0rd", false},
		{"lemah - hanya huruf", "Password", false},
		{"lemah - hanya angka dan huruf", "Passw0rd1", false},
		{"lemah - string kosong", "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := helper.IsStrongPassword(tc.input)
			if got != tc.want {
				t.Errorf("IsStrongPassword(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
