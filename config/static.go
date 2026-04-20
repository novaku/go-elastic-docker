package config

import "time"

const (
	DefaultJWTSecret         = "yOmMBRX5udUGEI6faHSBvbXhqT2bxAvWJALpwR/eG0k="
	DefaultJWTExpiryDuration = 24 * time.Hour
	DefaultAdminUsername     = "admin"
	DefaultAdminPassword     = "admin123"
	DefaultReadTimeout       = 10 * time.Second
	DefaultWriteTimeout      = 10 * time.Second
)
