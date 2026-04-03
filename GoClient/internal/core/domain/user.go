package domain

import "time"

type User struct {
	ID          string    `json:"id"`
	SubKeycloak string    `json:"sub_keycloak"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Nickname    string    `json:"nickname"`
	Email       string    `json:"email"`
	DateBirth   time.Time `json:"date_birth"`
	Gender      string    `json:"gender"`
}
