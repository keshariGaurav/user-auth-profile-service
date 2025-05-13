package structure

type EmailData struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	OTP     string `json:"otp,omitempty"`
	Link    string `json:"link,omitempty"`
}