package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	repo "family/internal/adapter/postgres/sqlc"
	"family/internal/token"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput       = errors.New("проверьте введённые данные")
	ErrWeakPassword       = errors.New("пароль должен быть не короче 8 символов")
	ErrEmailTaken         = errors.New("пользователь с таким email уже существует")
	ErrInvalidCredentials = errors.New("неверный email или пароль")
	ErrInvalidInvite      = errors.New("код приглашения недействителен или истёк")
	ErrInvalidRefresh     = errors.New("сессия истекла, войдите заново")
	ErrNotFound           = errors.New("пользователь не найден")
)

const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

var (
	colorRe = regexp.MustCompile(`^\d{1,3} \d{1,3}% \d{1,3}%$`)
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (Session, error)
	Join(ctx context.Context, req JoinRequest) (Session, error)
	Login(ctx context.Context, req LoginRequest) (Session, error)
	Refresh(ctx context.Context, refreshToken string) (Tokens, error)
	Logout(ctx context.Context, refreshToken string) error
	Me(ctx context.Context, userID uuid.UUID) (Profile, error)
}

type service struct {
	repo   *repo.Queries
	pool   *pgxpool.Pool
	tokens *token.Manager
}

func NewService(q *repo.Queries, pool *pgxpool.Pool, tokens *token.Manager) Service {
	return &service{repo: q, pool: pool, tokens: tokens}
}

type newUser struct {
	email, password, name, color string
}

func (u *newUser) normalize() error {
	u.email = strings.ToLower(strings.TrimSpace(u.email))
	u.name = strings.TrimSpace(u.name)
	u.color = strings.TrimSpace(u.color)
	if !emailRe.MatchString(u.email) || u.name == "" || !colorRe.MatchString(u.color) {
		return ErrInvalidInput
	}
	if len(u.password) < 8 {
		return ErrWeakPassword
	}
	return nil
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (Session, error) {
	u := newUser{req.Email, req.Password, req.Name, req.Color}
	familyName := strings.TrimSpace(req.FamilyName)
	if err := u.normalize(); err != nil {
		return Session{}, err
	}
	if familyName == "" {
		return Session{}, ErrInvalidInput
	}

	return s.createUser(ctx, u, func(q *repo.Queries, userID uuid.UUID) (uuid.UUID, string, error) {
		family, err := q.CreateFamily(ctx, familyName)
		if err != nil {
			return uuid.Nil, "", err
		}
		return family.ID, RoleOwner, q.AddFamilyMember(ctx, repo.AddFamilyMemberParams{
			UserID: userID, FamilyID: family.ID, Role: RoleOwner,
		})
	})
}

func (s *service) Join(ctx context.Context, req JoinRequest) (Session, error) {
	u := newUser{req.Email, req.Password, req.Name, req.Color}
	if err := u.normalize(); err != nil {
		return Session{}, err
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return Session{}, ErrInvalidInput
	}

	return s.createUser(ctx, u, func(q *repo.Queries, userID uuid.UUID) (uuid.UUID, string, error) {
		invite, err := q.GetActiveInvite(ctx, code)
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, "", ErrInvalidInvite
		}
		if err != nil {
			return uuid.Nil, "", err
		}
		// Атомарное «занять, только если свободен»: двое с одним кодом не пройдут
		n, err := q.UseInvite(ctx, code)
		if err != nil {
			return uuid.Nil, "", err
		}
		if n == 0 {
			return uuid.Nil, "", ErrInvalidInvite
		}
		return invite.FamilyID, RoleMember, q.AddFamilyMember(ctx, repo.AddFamilyMemberParams{
			UserID: userID, FamilyID: invite.FamilyID, Role: RoleMember,
		})
	})
}

// createUser в одной транзакции создаёт пользователя, вызывает attach для
// привязки к семье и выдаёт сессию.
func (s *service) createUser(
	ctx context.Context,
	u newUser,
	attach func(q *repo.Queries, userID uuid.UUID) (uuid.UUID, string, error),
) (Session, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
	if err != nil {
		return Session{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	q := s.repo.WithTx(tx)

	user, err := q.CreateUser(ctx, repo.CreateUserParams{
		Email: u.email, PasswordHash: string(hash), Name: u.name, Color: u.color,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Session{}, ErrEmailTaken
		}
		return Session{}, err
	}

	familyID, role, err := attach(q, user.ID)
	if err != nil {
		return Session{}, err
	}

	tokens, err := s.issue(ctx, q, user.ID)
	if err != nil {
		return Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}

	return Session{Tokens: tokens, User: profile(user, &familyID, &role)}, nil
}

func (s *service) Login(ctx context.Context, req LoginRequest) (Session, error) {
	user, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return Session{}, ErrInvalidCredentials
	}

	tokens, err := s.issue(ctx, s.repo, user.ID)
	if err != nil {
		return Session{}, err
	}
	p, err := s.Me(ctx, user.ID)
	if err != nil {
		return Session{}, err
	}
	return Session{Tokens: tokens, User: p}, nil
}

// Refresh — ротация: старый refresh-токен удаляется, выдаётся новая пара.
func (s *service) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	userID, err := s.repo.ConsumeRefreshToken(ctx, token.Hash(refreshToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return Tokens{}, ErrInvalidRefresh
	}
	if err != nil {
		return Tokens{}, err
	}
	return s.issue(ctx, s.repo, userID)
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	return s.repo.DeleteRefreshToken(ctx, token.Hash(refreshToken))
}

func (s *service) Me(ctx context.Context, userID uuid.UUID) (Profile, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	m, err := s.repo.GetMembership(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return profile(user, nil, nil), nil
	}
	if err != nil {
		return Profile{}, err
	}
	return profile(user, &m.FamilyID, &m.Role), nil
}

func (s *service) issue(ctx context.Context, q *repo.Queries, userID uuid.UUID) (Tokens, error) {
	access, err := s.tokens.NewAccess(userID)
	if err != nil {
		return Tokens{}, fmt.Errorf("sign access: %w", err)
	}
	raw, hash, err := token.NewOpaque()
	if err != nil {
		return Tokens{}, err
	}
	err = q.CreateRefreshToken(ctx, repo.CreateRefreshTokenParams{
		UserID: userID, TokenHash: hash, ExpiresAt: time.Now().Add(token.RefreshTTL),
	})
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: raw}, nil
}

func profile(u repo.User, familyID *uuid.UUID, role *string) Profile {
	return Profile{ID: u.ID, Email: u.Email, Name: u.Name, Color: u.Color, FamilyID: familyID, Role: role}
}
