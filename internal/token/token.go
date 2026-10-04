package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	AccessTTL  = 15 * time.Minute
	RefreshTTL = 30 * 24 * time.Hour
)

var ErrInvalid = errors.New("недействительный токен")

type Manager struct {
	secret []byte
}

func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

func (m *Manager) NewAccess(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(AccessTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *Manager) ParseAccess(raw string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return uuid.Nil, ErrInvalid
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalid
	}
	return id, nil
}

// NewOpaque возвращает случайную строку для клиента и её sha256 для хранения в БД.
func NewOpaque() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, Hash(raw), nil
}

func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
