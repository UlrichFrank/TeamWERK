package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DBPath         string
	JWTSecret      string
	BaseURL        string
	UploadDir      string
	FilesDir       string
	SMTP           SMTPConfig
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDEmail      string
	MailerDisabled  bool
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func Load() (*Config, error) {
	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	c := &Config{
		Port:      getEnv("PORT", "8080"),
		DBPath:    getEnv("DB_PATH", "./teamwerk.db"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		BaseURL:   getEnv("BASE_URL", "https://internal.team-stuttgart.org"),
		UploadDir: getEnv("UPLOAD_DIR", "./storage/uploads"),
		FilesDir:  getEnv("FILES_DIR", "./storage/files"),
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "mail.agenturserver.de"),
			Port:     smtpPort,
			User:     os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASS"),
			From:     getEnv("SMTP_FROM", "TeamWERK <vorstand@team-stuttgart.org>"),
		},
		VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDEmail:      getEnv("VAPID_EMAIL", "vorstand@team-stuttgart.org"),
		MailerDisabled:  os.Getenv("MAILER_DISABLED") == "true",
	}
	if c.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be set")
	}
	return c, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
