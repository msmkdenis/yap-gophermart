package model

import "time"

type Order struct {
	ID         string    `db:"id"`
	Number     string    `db:"number"`
	UserLogin  string    `db:"user_login"`
	UploadedAt time.Time `db:"uploaded_at"`
	Status     string    `db:"status"`
}
