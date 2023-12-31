package config

type LogConfig struct {
	Output   string             `yaml:",omitempty" json:"output,omitempty"`
	Level    string             `yaml:",omitempty" json:"level,omitempty"`
	Format   string             `yaml:",omitempty" json:"format,omitempty"`
	Rotation *LogRotationConfig `yaml:",omitempty" json:"rotation,omitempty"`
}

type LogRotationConfig struct {
	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int `yaml:"maxSize,omitempty" json:"maxSize,omitempty"`
	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int `yaml:"maxAge,omitempty" json:"maxAge,omitempty"`
	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `yaml:"maxBackups,omitempty" json:"maxBackups,omitempty"`
	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time. The default is to use UTC
	// time.
	LocalTime bool `yaml:"localTime,omitempty" json:"localTime,omitempty"`
	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `yaml:"compress,omitempty" json:"compress,omitempty"`
}

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
