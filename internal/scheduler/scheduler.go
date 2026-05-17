package scheduler

import (
	"database/sql"
	"log"
)

type Scheduler struct{ db *sql.DB }

func New(db *sql.DB) *Scheduler { return &Scheduler{db: db} }

func (s *Scheduler) Run() {
	s.cleanExpiredTokens()
}

func (s *Scheduler) cleanExpiredTokens() {
	res, err := s.db.Exec(
		`DELETE FROM invitation_tokens WHERE expires_at < CURRENT_TIMESTAMP AND used_at IS NULL;
		 DELETE FROM password_reset_tokens WHERE expires_at < CURRENT_TIMESTAMP AND used_at IS NULL;
		 DELETE FROM refresh_tokens WHERE expires_at < CURRENT_TIMESTAMP;`)
	if err != nil {
		log.Printf("scheduler: cleanup error: %v", err)
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		log.Printf("scheduler: cleaned %d expired tokens", n)
	}
}
