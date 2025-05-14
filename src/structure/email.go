package structure

type EmailData struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Data     map[string]string `json:"data" validate:"required"`
	Link    string `json:"link,omitempty"`
	Template string `json:"template,omitempty"`
}