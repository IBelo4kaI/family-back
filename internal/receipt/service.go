package receipt

import (
	"context"
	"errors"
	"strings"
	"time"

	repo "family/internal/adapter/postgres/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidInput = errors.New("проверьте введённые данные")
	ErrDuplicate    = errors.New("этот чек уже добавлен")
	ErrNotFound     = errors.New("запись не найдена")
)

type Actor struct {
	UserID   uuid.UUID
	FamilyID uuid.UUID
}

type Service interface {
	Check(ctx context.Context, qrraw string) (Receipt, error)
	Save(ctx context.Context, a Actor, in SaveInput) (Transaction, error)
	Items(ctx context.Context, a Actor, transactionID uuid.UUID) ([]StoredItem, error)
}

type service struct {
	repo    *repo.Queries
	pool    *pgxpool.Pool
	checker *checker
}

func NewService(q *repo.Queries, pool *pgxpool.Pool, tokens []string) Service {
	return &service{repo: q, pool: pool, checker: newChecker(tokens)}
}

func (s *service) Check(ctx context.Context, qrraw string) (Receipt, error) {
	if strings.TrimSpace(qrraw) == "" {
		return Receipt{}, ErrInvalidInput
	}
	raw, err := s.checker.check(ctx, strings.TrimSpace(qrraw))
	if err != nil {
		return Receipt{}, err
	}
	return toReceipt(raw)
}

// Save создаёт транзакцию-расход по чеку вместе с позициями в одной транзакции БД.
func (s *service) Save(ctx context.Context, a Actor, in SaveInput) (Transaction, error) {
	date, err := time.Parse("2006-01-02", in.Date)
	if err != nil || in.TotalSum <= 0 || strings.TrimSpace(in.Key) == "" ||
		(in.Scope != "personal" && in.Scope != "family") || len(in.Items) == 0 {
		return Transaction{}, ErrInvalidInput
	}
	for _, it := range in.Items {
		if strings.TrimSpace(it.Name) == "" || it.Price < 0 || it.Sum < 0 || it.Quantity <= 0 {
			return Transaction{}, ErrInvalidInput
		}
	}

	cat, err := s.repo.GetCategory(ctx, repo.GetCategoryParams{ID: in.CategoryID, FamilyID: pgUUID(a.FamilyID)})
	if err != nil || cat.Kind != "expense" {
		return Transaction{}, ErrInvalidInput
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Transaction{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := s.repo.WithTx(tx)
	row, err := q.CreateReceiptTransaction(ctx, repo.CreateReceiptTransactionParams{
		FamilyID: a.FamilyID, UserID: a.UserID, Amount: in.TotalSum, Date: date,
		CategoryID: in.CategoryID, Scope: in.Scope,
		SellerInn: optional(in.SellerINN), SellerName: optional(in.SellerName), ReceiptKey: &in.Key,
	})
	if err != nil {
		return Transaction{}, mapDBError(err)
	}
	for i, it := range in.Items {
		if err := q.CreateTransactionItem(ctx, repo.CreateTransactionItemParams{
			TransactionID: row.ID, Position: int32(i), Name: strings.TrimSpace(it.Name),
			Price: it.Price, Quantity: it.Quantity, Sum: it.Sum,
		}); err != nil {
			return Transaction{}, mapDBError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, err
	}

	return Transaction{
		ID: row.ID, Amount: row.Amount, Date: row.Date.Format("2006-01-02"), CategoryID: row.CategoryID,
		Type: row.Type, Scope: row.Scope, AuthorID: row.AuthorID, Source: row.Source,
	}, nil
}

func (s *service) Items(ctx context.Context, a Actor, transactionID uuid.UUID) ([]StoredItem, error) {
	rows, err := s.repo.ListTransactionItems(ctx, repo.ListTransactionItemsParams{
		TransactionID: transactionID, FamilyID: a.FamilyID, UserID: a.UserID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]StoredItem, len(rows))
	for i, r := range rows {
		out[i] = StoredItem{
			ID: r.ID, Position: r.Position, Name: r.Name, Price: r.Price,
			Quantity: r.Quantity, Sum: r.Sum, CategoryID: r.CategoryID,
		}
	}
	return out, nil
}

func optional(s string) *string {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	return &s
}

func mapDBError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrDuplicate
		case "23503", "23514":
			return ErrInvalidInput
		}
	}
	return err
}

func pgUUID(id uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: id, Valid: true} }
