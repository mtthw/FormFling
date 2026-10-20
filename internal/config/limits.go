package config

import "time"

const (
	MaxRequestBodyBytes   int64 = 1 << 20 // 1 MiB
	HTTPReadHeaderTimeout       = 5 * time.Second
	HTTPReadTimeout             = 15 * time.Second
	HTTPWriteTimeout            = 60 * time.Second
	HTTPIdleTimeout             = 60 * time.Second
	SMTPConnectTimeout          = 5 * time.Second
	SMTPTimeout                 = 20 * time.Second
)
