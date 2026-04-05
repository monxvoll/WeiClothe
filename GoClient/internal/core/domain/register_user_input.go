package domain

import "time"

type RegisterUserInput struct {
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	DateBirth time.Time `json:"date_birth"`
	Gender    string    `json:"gender"`
}
