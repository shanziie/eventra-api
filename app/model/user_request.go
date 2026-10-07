package model

// PutUserRequest memuat data pembaruan penuh profil pengguna.
type PutUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,username"`
	Email    string `json:"email" validate:"required,email,max=120"`
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	IsActive *bool  `json:"is_active" validate:"required"`
}

// PatchUserRequest memuat data pembaruan sebagian profil pengguna dengan pointer + omitnil.
type PatchUserRequest struct {
	Username *string `json:"username,omitempty" validate:"omitnil,min=3,max=30,username"`
	Email    *string `json:"email,omitempty" validate:"omitnil,email,max=120"`
	FullName *string `json:"full_name,omitempty" validate:"omitnil,min=2,max=100"`
	IsActive *bool   `json:"is_active,omitempty" validate:"omitnil"`
}

// AssignRoleRequest memuat role baru untuk penugasan role pengguna (BR-U4).
type AssignRoleRequest struct {
	Role string `json:"role" validate:"required"`
}
