package structure

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type VerifyOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

// Add ResetRequest struct for ResetPassword functionality
// Used in ResetPassword controller
// Accepts email, token, password, confirmPassword
// All fields required
//
type ResetRequest struct {
	Email           string `json:"email" validate:"required,email"`
	Token           string `json:"token" validate:"required"`
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirmPassword" validate:"required,min=8"`
}

// Add Request struct for UpdatePassword functionality
type UpdatePasswordRequest struct {
	Email           string `json:"email" validate:"required,email"`
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8"`
}