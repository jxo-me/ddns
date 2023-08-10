package config

// Webhook Webhook
type Webhook struct {
	WebhookURL         string `json:"webhookURL"`
	WebhookRequestBody string `json:"webhookRequestBody"`
	WebhookHeaders     string `json:"webhookHeaders"`
}
