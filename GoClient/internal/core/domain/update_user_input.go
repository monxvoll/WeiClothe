package domain

import "time"

type UpdateUserInput struct {
	SubKeycloak string    `json:"sub_keycloak"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Nickname    string    `json:"nickname"`
	DateBirth   time.Time `json:"date_birth"`
	Gender      string    `json:"gender"`
}
