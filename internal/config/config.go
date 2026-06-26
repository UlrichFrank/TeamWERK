package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port            string
	DBPath          string
	JWTSecret       string
	BaseURL         string
	UploadDir       string
	FilesDir        string
	BeitragslaufDir string
	SMTP            SMTPConfig
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDEmail      string
	MailerDisabled  bool
	// MetricsToken schützt GET /api/metrics. Leer ⇒ Endpoint deaktiviert (404).
	MetricsToken string
	// LogFormat steuert den slog-Handler: "json" (Default, Prod) oder "text" (lokal).
	LogFormat string
	// AuthRateLimitPerMin: erlaubte Anfragen pro Minute und Client-IP auf die
	// unauthentifizierten Auth-Routen (login/refresh/forgot-password/reset-password).
	// 0 deaktiviert das IP-Rate-Limiting (z.B. in Tests).
	AuthRateLimitPerMin int
	// LoginMaxFailures: aufeinanderfolgende Login-Fehlversuche, ab denen ein Konto
	// gesperrt wird. 0 deaktiviert den Account-Lockout (z.B. in Tests).
	LoginMaxFailures int
	// LoginLockMinutes: Dauer der Konto-Sperre in Minuten nach Überschreiten von
	// LoginMaxFailures.
	LoginLockMinutes int
	// ForgotPasswordCooldownSec: Mindestabstand in Sekunden zwischen zwei
	// Passwort-Reset-Mails pro Konto (zusätzlich zur IP-Drosselung). 0 deaktiviert
	// die Konto-Drosselung (z.B. in Tests).
	ForgotPasswordCooldownSec int
	// PasswordMinLength: serverseitig erzwungene Mindestlänge für neue Passwörter
	// (Register/Reset/Change). <=0 wird als Default 12 behandelt.
	PasswordMinLength int
	// HSTSEnabled steuert den Strict-Transport-Security-Header. Default false —
	// erst nach Live-Zertifikat aktivieren (sonst Aussperrung bei fehlendem TLS).
	HSTSEnabled bool
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
		Port:            getEnv("PORT", "8080"),
		DBPath:          getEnv("DB_PATH", "./teamwerk.db"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		BaseURL:         getEnv("BASE_URL", "https://internal.team-stuttgart.org"),
		UploadDir:       getEnv("UPLOAD_DIR", "./storage/uploads"),
		FilesDir:        getEnv("FILES_DIR", "./storage/files"),
		BeitragslaufDir: getEnv("BEITRAGSLAUF_DIR", "./storage/beitragslauf-protokolle"),
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
		MetricsToken:    os.Getenv("METRICS_TOKEN"),
		LogFormat:       getEnv("LOG_FORMAT", "json"),

		AuthRateLimitPerMin:       getEnvInt("AUTH_RATE_LIMIT_PER_MIN", 10),
		LoginMaxFailures:          getEnvInt("LOGIN_MAX_FAILURES", 5),
		LoginLockMinutes:          getEnvInt("LOGIN_LOCK_MINUTES", 15),
		ForgotPasswordCooldownSec: getEnvInt("FORGOT_PASSWORD_COOLDOWN_SEC", 60),
		PasswordMinLength:         getEnvInt("PASSWORD_MIN_LENGTH", 12),
		HSTSEnabled:               os.Getenv("HSTS_ENABLED") == "true",
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

// getEnvInt liest einen ganzzahligen Env-Wert; bei fehlendem oder ungültigem
// Wert wird der Fallback verwendet.
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
