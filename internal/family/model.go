package family

import (
	"time"

	"github.com/google/uuid"
)

type Member struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Color    string    `json:"color"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
}

type Family struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Members   []Member  `json:"members"`
}

type Invite struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}
