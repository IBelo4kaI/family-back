package auth

import (
	"time"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	FamilyName string `json:"familyName"`
}

type JoinRequest struct {
	Code     string `json:"code"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Color    string `json:"color"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type Profile struct {
	ID       uuid.UUID  `json:"id"`
	Email    string     `json:"email"`
	Name     string     `json:"name"`
	Color    string     `json:"color"`
	FamilyID *uuid.UUID `json:"familyId"`
	Role     *string    `json:"role"`
}

type Session struct {
	Tokens
	User Profile `json:"user"`
}

type Invite struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}
