package config

// Webhook Webhook
type Webhook struct {
	// 支持的变量 #{ipv4Addr}=新的IPv4地址,
	// #{ipv4Result}=IPv4地址更新结果: 未改变 失败 成功,
	// #{ipv4Domains}=IPv4的域名，多个以,分割,
	// #{ipv6Addr}=新的IPv6地址,
	// #{ipv6Result}=IPv6地址更新结果: 未改变 失败 成功,
	// #{ipv6Domains}=IPv6的域名，多个以,分割
	WebhookURL string `json:"webhookURL"`
	// 如 RequestBody 为空则为 GET 请求，否则为 POST 请求。支持的变量同上
	WebhookRequestBody string `json:"webhookRequestBody"`
	// 一行一个Header, 如：Authorization: Bearer API_KEY
	WebhookHeaders string `json:"webhookHeaders"`
}
