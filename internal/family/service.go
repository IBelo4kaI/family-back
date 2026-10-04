package family

import (
	"context"
	"errors"
	"time"

	repo "family/internal/adapter/postgres/sqlc"
	"family/internal/auth"
	"family/internal/token"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotOwner   = errors.New("это действие доступно только владельцу семьи")
	ErrNotFound   = errors.New("участник не найден")
	ErrCannotKick = errors.New("владельца нельзя исключить, он может только покинуть семью")
	ErrNoFamily   = errors.New("вы не состоите в семье")
)

const inviteTTL = 7 * 24 * time.Hour

type Service interface {
	Get(ctx context.Context, familyID uuid.UUID) (Family, error)
	CreateInvite(ctx context.Context, familyID, userID uuid.UUID) (Invite, error)
	Kick(ctx context.Context, familyID, ownerID, memberID uuid.UUID) error
	Leave(ctx context.Context, familyID, userID uuid.UUID) error
}

type service struct {
	repo *repo.Queries
	pool *pgxpool.Pool
}

func NewService(q *repo.Queries, pool *pgxpool.Pool) Service {
	return &service{repo: q, pool: pool}
}

func (s *service) Get(ctx context.Context, familyID uuid.UUID) (Family, error) {
	f, err := s.repo.GetFamily(ctx, familyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Family{}, ErrNoFamily
	}
	if err != nil {
		return Family{}, err
	}
	rows, err := s.repo.ListFamilyMembers(ctx, familyID)
	if err != nil {
		return Family{}, err
	}
	members := make([]Member, len(rows))
	for i, r := range rows {
		members[i] = Member{ID: r.ID, Name: r.Name, Color: r.Color, Role: r.Role, JoinedAt: r.JoinedAt}
	}
	return Family{ID: f.ID, Name: f.Name, CreatedAt: f.CreatedAt, Members: members}, nil
}

func (s *service) CreateInvite(ctx context.Context, familyID, userID uuid.UUID) (Invite, error) {
	if err := s.requireOwner(ctx, familyID, userID); err != nil {
		return Invite{}, err
	}
	code, _, err := token.NewOpaque()
	if err != nil {
		return Invite{}, err
	}
	expires := time.Now().Add(inviteTTL)
	err = s.repo.CreateInvite(ctx, repo.CreateInviteParams{
		Code: code, FamilyID: familyID, CreatedBy: userID, ExpiresAt: expires,
	})
	if err != nil {
		return Invite{}, err
	}
	return Invite{Code: code, ExpiresAt: expires}, nil
}

func (s *service) Kick(ctx context.Context, familyID, ownerID, memberID uuid.UUID) error {
	if err := s.requireOwner(ctx, familyID, ownerID); err != nil {
		return err
	}
	if ownerID == memberID {
		return ErrCannotKick
	}
	return s.remove(ctx, familyID, memberID)
}

func (s *service) Leave(ctx context.Context, familyID, userID uuid.UUID) error {
	return s.remove(ctx, familyID, userID)
}

// remove убирает участника, а если он был владельцем — передаёт роль
// старейшему по дате вступления; последний участник забирает семью с собой.
func (s *service) remove(ctx context.Context, familyID, userID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := s.repo.WithTx(tx)

	m, err := q.GetMembership(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && m.FamilyID != familyID) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if _, err := q.RemoveFamilyMember(ctx, repo.RemoveFamilyMemberParams{UserID: userID, FamilyID: familyID}); err != nil {
		return err
	}
	if err := q.DeleteUserRefreshTokens(ctx, userID); err != nil {
		return err
	}

	next, err := q.OldestMember(ctx, familyID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if err := q.DeleteFamily(ctx, familyID); err != nil {
			return err
		}
	case err != nil:
		return err
	case m.Role == auth.RoleOwner:
		if err := q.SetMemberRole(ctx, repo.SetMemberRoleParams{UserID: next, FamilyID: familyID, Role: auth.RoleOwner}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *service) requireOwner(ctx context.Context, familyID, userID uuid.UUID) error {
	m, err := s.repo.GetMembership(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (m.FamilyID != familyID || m.Role != auth.RoleOwner)) {
		return ErrNotOwner
	}
	return err
}
