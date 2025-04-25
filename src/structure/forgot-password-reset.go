package structure
type ForgotPasswordRequest struct {
	Username string `json:"username" validate:"required"`
}
