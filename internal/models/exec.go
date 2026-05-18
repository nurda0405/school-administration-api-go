package models

import "database/sql"

type Exec struct {
	ID                  int
	FirstName           string
	LastName            string
	Email               string
	Username            string
	Password            string
	PasswordChangedAt   sql.NullString
	UserCreatedAt       sql.NullString
	PasswordResetCode   sql.NullString
	PasswordCodeExpires sql.NullString
	InactiveStatus      bool
	Role                string
}
