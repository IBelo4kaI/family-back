package budget

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
	ErrNotFound     = errors.New("запись не найдена")
	ErrClosed       = errors.New("платёж уже закрыт")
	ErrLimitExists  = errors.New("для этой категории лимит уже задан")
)

const (
	actionConfirm = "confirm"
	actionSkip    = "skip"
)

// Actor — кто и в рамках какой семьи выполняет операцию.
type Actor struct {
	UserID   uuid.UUID
	FamilyID uuid.UUID
}

type Service interface {
	Load(ctx context.Context, a Actor) (Data, error)

	CreateTransaction(ctx context.Context, a Actor, in TransactionInput) (Transaction, error)
	UpdateTransaction(ctx context.Context, a Actor, id uuid.UUID, in TransactionInput) (Transaction, error)
	DeleteTransaction(ctx context.Context, a Actor, id uuid.UUID) error

	CreatePayment(ctx context.Context, a Actor, in PaymentInput) (RecurringPayment, error)
	UpdatePayment(ctx context.Context, a Actor, id uuid.UUID, in PaymentInput) (RecurringPayment, error)
	ClosePayment(ctx context.Context, a Actor, id uuid.UUID) error
	SettlePayment(ctx context.Context, a Actor, id uuid.UUID, action string) (SettleResult, error)

	CreateCard(ctx context.Context, a Actor, in CardInput) (CreditCard, error)
	UpdateCard(ctx context.Context, a Actor, id uuid.UUID, in CardInput) (CreditCard, error)
	CloseCard(ctx context.Context, a Actor, id uuid.UUID) error
	SettleCard(ctx context.Context, a Actor, id uuid.UUID, action string) (SettleResult, error)

	CreateLimit(ctx context.Context, a Actor, in LimitInput) (CategoryLimit, error)
	UpdateLimit(ctx context.Context, a Actor, id uuid.UUID, in LimitInput) (CategoryLimit, error)
	DeleteLimit(ctx context.Context, a Actor, id uuid.UUID) error

	CreateGoal(ctx context.Context, a Actor, in GoalInput) (SavingsGoal, error)
	UpdateGoal(ctx context.Context, a Actor, id uuid.UUID, in GoalInput) (SavingsGoal, error)
	DeleteGoal(ctx context.Context, a Actor, id uuid.UUID) error
	TopUpGoal(ctx context.Context, a Actor, id uuid.UUID, amount int64) (SavingsGoal, error)
}

type service struct {
	repo *repo.Queries
	pool *pgxpool.Pool
}

func NewService(q *repo.Queries, pool *pgxpool.Pool) Service {
	return &service{repo: q, pool: pool}
}

func (s *service) Load(ctx context.Context, a Actor) (Data, error) {
	cats, err := s.repo.ListCategories(ctx, pgtype.UUID{Bytes: a.FamilyID, Valid: true})
	if err != nil {
		return Data{}, err
	}
	txs, err := s.repo.ListTransactions(ctx, repo.ListTransactionsParams{FamilyID: a.FamilyID, UserID: a.UserID})
	if err != nil {
		return Data{}, err
	}
	payments, err := s.repo.ListPayments(ctx, repo.ListPaymentsParams{FamilyID: a.FamilyID, UserID: a.UserID})
	if err != nil {
		return Data{}, err
	}
	cards, err := s.repo.ListCards(ctx, repo.ListCardsParams{FamilyID: a.FamilyID, UserID: a.UserID})
	if err != nil {
		return Data{}, err
	}
	limits, err := s.repo.ListLimits(ctx, repo.ListLimitsParams{FamilyID: a.FamilyID, UserID: a.UserID})
	if err != nil {
		return Data{}, err
	}
	goals, err := s.repo.ListGoals(ctx, repo.ListGoalsParams{FamilyID: a.FamilyID, UserID: a.UserID})
	if err != nil {
		return Data{}, err
	}

	return Data{
		Categories:   mapSlice(cats, func(c repo.ListCategoriesRow) Category { return Category(c) }),
		Transactions: mapSlice(txs, toTransaction),
		Payments:     mapSlice(payments, toPayment),
		Cards:        mapSlice(cards, toCard),
		Limits:       mapSlice(limits, toLimit),
		Goals:        mapSlice(goals, toGoal),
	}, nil
}

// --- Транзакции ---

func (s *service) CreateTransaction(ctx context.Context, a Actor, in TransactionInput) (Transaction, error) {
	p, err := s.validateTransaction(ctx, a, in)
	if err != nil {
		return Transaction{}, err
	}
	p.FamilyID, p.UserID, p.Source = a.FamilyID, a.UserID, "manual"
	t, err := s.repo.CreateTransaction(ctx, p)
	if err != nil {
		return Transaction{}, mapDBError(err)
	}
	return toTransaction(t), nil
}

func (s *service) UpdateTransaction(ctx context.Context, a Actor, id uuid.UUID, in TransactionInput) (Transaction, error) {
	p, err := s.validateTransaction(ctx, a, in)
	if err != nil {
		return Transaction{}, err
	}
	t, err := s.repo.UpdateTransaction(ctx, repo.UpdateTransactionParams{
		Amount: p.Amount, Date: p.Date, CategoryID: p.CategoryID, Type: p.Type, Scope: p.Scope,
		ID: id, FamilyID: a.FamilyID, UserID: a.UserID,
	})
	if err != nil {
		return Transaction{}, mapDBError(err)
	}
	return toTransaction(t), nil
}

func (s *service) DeleteTransaction(ctx context.Context, a Actor, id uuid.UUID) error {
	n, err := s.repo.DeleteTransaction(ctx, repo.DeleteTransactionParams{ID: id, FamilyID: a.FamilyID, UserID: a.UserID})
	return affected(n, err)
}

func (s *service) validateTransaction(ctx context.Context, a Actor, in TransactionInput) (repo.CreateTransactionParams, error) {
	date, err := parseDate(in.Date)
	if err != nil || in.Amount <= 0 || !validScope(in.Scope) || !oneOf(in.Type, "income", "expense") {
		return repo.CreateTransactionParams{}, ErrInvalidInput
	}
	if err := s.checkCategory(ctx, a, in.CategoryID, in.Type); err != nil {
		return repo.CreateTransactionParams{}, err
	}
	return repo.CreateTransactionParams{
		Amount: in.Amount, Date: date, CategoryID: in.CategoryID, Type: in.Type, Scope: in.Scope,
	}, nil
}

// --- Регулярные платежи ---

func (s *service) CreatePayment(ctx context.Context, a Actor, in PaymentInput) (RecurringPayment, error) {
	p, err := s.validatePayment(ctx, a, in)
	if err != nil {
		return RecurringPayment{}, err
	}
	p.FamilyID, p.UserID = a.FamilyID, a.UserID
	row, err := s.repo.CreatePayment(ctx, p)
	if err != nil {
		return RecurringPayment{}, mapDBError(err)
	}
	return toPayment(row), nil
}

func (s *service) UpdatePayment(ctx context.Context, a Actor, id uuid.UUID, in PaymentInput) (RecurringPayment, error) {
	p, err := s.validatePayment(ctx, a, in)
	if err != nil {
		return RecurringPayment{}, err
	}
	row, err := s.repo.UpdatePayment(ctx, repo.UpdatePaymentParams{
		Name: p.Name, Kind: p.Kind, Amount: p.Amount, Period: p.Period, NextDate: p.NextDate,
		CategoryID: p.CategoryID, NotifyDaysBefore: p.NotifyDaysBefore, Scope: p.Scope,
		TotalAmount: p.TotalAmount, EndDate: p.EndDate,
		ID: id, FamilyID: a.FamilyID, UserID: a.UserID,
	})
	if err != nil {
		return RecurringPayment{}, mapDBError(err)
	}
	return toPayment(row), nil
}

func (s *service) ClosePayment(ctx context.Context, a Actor, id uuid.UUID) error {
	n, err := s.repo.ClosePayment(ctx, repo.ClosePaymentParams{ID: id, FamilyID: a.FamilyID, UserID: a.UserID})
	return affected(n, err)
}

func (s *service) validatePayment(ctx context.Context, a Actor, in PaymentInput) (repo.CreatePaymentParams, error) {
	name := strings.TrimSpace(in.Name)
	next, err := parseDate(in.NextDate)
	if err != nil || name == "" || in.Amount <= 0 || in.NotifyDaysBefore < 0 || !validScope(in.Scope) ||
		!oneOf(in.Kind, "subscription", "loan") || !oneOf(in.Period, "month", "year") ||
		(in.TotalAmount != nil && *in.TotalAmount <= 0) {
		return repo.CreatePaymentParams{}, ErrInvalidInput
	}
	var end *time.Time
	if in.EndDate != nil && *in.EndDate != "" {
		d, err := parseDate(*in.EndDate)
		if err != nil {
			return repo.CreatePaymentParams{}, ErrInvalidInput
		}
		end = &d
	}
	if err := s.checkCategory(ctx, a, in.CategoryID, "expense"); err != nil {
		return repo.CreatePaymentParams{}, err
	}
	p := repo.CreatePaymentParams{
		Name: name, Kind: in.Kind, Amount: in.Amount, Period: in.Period, NextDate: next,
		CategoryID: in.CategoryID, NotifyDaysBefore: in.NotifyDaysBefore, Scope: in.Scope, EndDate: end,
	}
	// Сумма и дата окончания имеют смысл только для кредита
	if in.Kind == "loan" {
		p.TotalAmount = in.TotalAmount
	} else {
		p.EndDate = nil
	}
	return p, nil
}

// --- Кредитные карты ---

func (s *service) CreateCard(ctx context.Context, a Actor, in CardInput) (CreditCard, error) {
	p, err := s.validateCard(ctx, a, in)
	if err != nil {
		return CreditCard{}, err
	}
	p.FamilyID, p.UserID = a.FamilyID, a.UserID
	row, err := s.repo.CreateCard(ctx, p)
	if err != nil {
		return CreditCard{}, mapDBError(err)
	}
	return toCard(row), nil
}

func (s *service) UpdateCard(ctx context.Context, a Actor, id uuid.UUID, in CardInput) (CreditCard, error) {
	p, err := s.validateCard(ctx, a, in)
	if err != nil {
		return CreditCard{}, err
	}
	row, err := s.repo.UpdateCard(ctx, repo.UpdateCardParams{
		Bank: p.Bank, AmountDue: p.AmountDue, DueDate: p.DueDate, CategoryID: p.CategoryID,
		NotifyDaysBefore: p.NotifyDaysBefore, Scope: p.Scope, CreditLimit: p.CreditLimit,
		ID: id, FamilyID: a.FamilyID, UserID: a.UserID,
	})
	if err != nil {
		return CreditCard{}, mapDBError(err)
	}
	return toCard(row), nil
}

func (s *service) CloseCard(ctx context.Context, a Actor, id uuid.UUID) error {
	n, err := s.repo.CloseCard(ctx, repo.CloseCardParams{ID: id, FamilyID: a.FamilyID, UserID: a.UserID})
	return affected(n, err)
}

func (s *service) validateCard(ctx context.Context, a Actor, in CardInput) (repo.CreateCardParams, error) {
	bank := strings.TrimSpace(in.Bank)
	due, err := parseDate(in.DueDate)
	if err != nil || bank == "" || in.AmountDue < 0 || in.NotifyDaysBefore < 0 || !validScope(in.Scope) ||
		(in.CreditLimit != nil && *in.CreditLimit <= 0) {
		return repo.CreateCardParams{}, ErrInvalidInput
	}
	if err := s.checkCategory(ctx, a, in.CategoryID, "expense"); err != nil {
		return repo.CreateCardParams{}, err
	}
	return repo.CreateCardParams{
		Bank: bank, AmountDue: in.AmountDue, DueDate: due, CategoryID: in.CategoryID,
		NotifyDaysBefore: in.NotifyDaysBefore, Scope: in.Scope, CreditLimit: in.CreditLimit,
	}, nil
}

// --- Подтверждение платежа ---

func (s *service) SettlePayment(ctx context.Context, a Actor, id uuid.UUID, action string) (SettleResult, error) {
	if !oneOf(action, actionConfirm, actionSkip) {
		return SettleResult{}, ErrInvalidInput
	}
	return s.settle(ctx, func(q *repo.Queries) (settleTarget, error) {
		p, err := q.GetPaymentForUpdate(ctx, repo.GetPaymentForUpdateParams{ID: id, FamilyID: a.FamilyID, UserID: a.UserID})
		if err != nil {
			return settleTarget{}, err
		}
		months := 1
		if p.Period == "year" {
			months = 12
		}
		next := addMonths(p.NextDate, months)
		return settleTarget{
			active: p.Status == "active", amount: p.Amount, categoryID: p.CategoryID, scope: p.Scope, next: next,
			save: func() error {
				return q.SetPaymentNextDate(ctx, repo.SetPaymentNextDateParams{ID: id, NextDate: next})
			},
		}, nil
	}, a, action)
}

func (s *service) SettleCard(ctx context.Context, a Actor, id uuid.UUID, action string) (SettleResult, error) {
	if !oneOf(action, actionConfirm, actionSkip) {
		return SettleResult{}, ErrInvalidInput
	}
	return s.settle(ctx, func(q *repo.Queries) (settleTarget, error) {
		c, err := q.GetCardForUpdate(ctx, repo.GetCardForUpdateParams{ID: id, FamilyID: a.FamilyID, UserID: a.UserID})
		if err != nil {
			return settleTarget{}, err
		}
		next := addMonths(c.DueDate, 1)
		return settleTarget{
			active: c.Status == "active", amount: c.AmountDue, categoryID: c.CategoryID, scope: c.Scope, next: next,
			save: func() error {
				return q.SetCardDueDate(ctx, repo.SetCardDueDateParams{ID: id, DueDate: next})
			},
		}, nil
	}, a, action)
}

type settleTarget struct {
	active     bool
	amount     int64
	categoryID string
	scope      string
	next       time.Time
	save       func() error
}

// settle двигает дату платежа на период вперёд; при confirm в той же
// транзакции создаётся расход на сегодня. Строка блокируется FOR UPDATE —
// повторное подтверждение не создаст дубль.
func (s *service) settle(
	ctx context.Context,
	load func(q *repo.Queries) (settleTarget, error),
	a Actor,
	action string,
) (SettleResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SettleResult{}, err
	}
	defer tx.Rollback(ctx)
	q := s.repo.WithTx(tx)

	t, err := load(q)
	if err != nil {
		return SettleResult{}, mapDBError(err)
	}
	if !t.active {
		return SettleResult{}, ErrClosed
	}
	if err := t.save(); err != nil {
		return SettleResult{}, err
	}

	res := SettleResult{NextDate: fmtDate(t.next)}
	if action == actionConfirm {
		row, err := q.CreateTransaction(ctx, repo.CreateTransactionParams{
			FamilyID: a.FamilyID, UserID: a.UserID, Amount: t.amount, Date: today(),
			CategoryID: t.categoryID, Type: "expense", Scope: t.scope, Source: "manual",
		})
		if err != nil {
			return SettleResult{}, mapDBError(err)
		}
		tr := toTransaction(row)
		res.Transaction = &tr
	}
	if err := tx.Commit(ctx); err != nil {
		return SettleResult{}, err
	}
	return res, nil
}

// --- Лимиты ---

func (s *service) CreateLimit(ctx context.Context, a Actor, in LimitInput) (CategoryLimit, error) {
	if err := s.validateLimit(ctx, a, in); err != nil {
		return CategoryLimit{}, err
	}
	row, err := s.repo.CreateLimit(ctx, repo.CreateLimitParams{
		CategoryID: in.CategoryID, LimitAmount: in.Limit, Scope: in.Scope, FamilyID: a.FamilyID, UserID: a.UserID,
	})
	if err != nil {
		return CategoryLimit{}, mapDBError(err)
	}
	return toLimit(row), nil
}

func (s *service) UpdateLimit(ctx context.Context, a Actor, id uuid.UUID, in LimitInput) (CategoryLimit, error) {
	if err := s.validateLimit(ctx, a, in); err != nil {
		return CategoryLimit{}, err
	}
	row, err := s.repo.UpdateLimit(ctx, repo.UpdateLimitParams{
		CategoryID: in.CategoryID, LimitAmount: in.Limit, Scope: in.Scope, ID: id, FamilyID: a.FamilyID, UserID: a.UserID,
	})
	if err != nil {
		return CategoryLimit{}, mapDBError(err)
	}
	return toLimit(row), nil
}

func (s *service) DeleteLimit(ctx context.Context, a Actor, id uuid.UUID) error {
	n, err := s.repo.DeleteLimit(ctx, repo.DeleteLimitParams{ID: id, FamilyID: a.FamilyID, UserID: a.UserID})
	return affected(n, err)
}

func (s *service) validateLimit(ctx context.Context, a Actor, in LimitInput) error {
	if in.Limit <= 0 || !validScope(in.Scope) {
		return ErrInvalidInput
	}
	return s.checkCategory(ctx, a, in.CategoryID, "expense")
}

// --- Цели ---

func (s *service) CreateGoal(ctx context.Context, a Actor, in GoalInput) (SavingsGoal, error) {
	p, err := validateGoal(in)
	if err != nil {
		return SavingsGoal{}, err
	}
	row, err := s.repo.CreateGoal(ctx, repo.CreateGoalParams{
		Name: p.Name, Target: p.Target, Saved: p.Saved, Deadline: p.Deadline, Scope: p.Scope,
		FamilyID: a.FamilyID, UserID: a.UserID,
	})
	if err != nil {
		return SavingsGoal{}, mapDBError(err)
	}
	return toGoal(row), nil
}

func (s *service) UpdateGoal(ctx context.Context, a Actor, id uuid.UUID, in GoalInput) (SavingsGoal, error) {
	p, err := validateGoal(in)
	if err != nil {
		return SavingsGoal{}, err
	}
	row, err := s.repo.UpdateGoal(ctx, repo.UpdateGoalParams{
		Name: p.Name, Target: p.Target, Saved: p.Saved, Deadline: p.Deadline, Scope: p.Scope,
		ID: id, FamilyID: a.FamilyID, UserID: a.UserID,
	})
	if err != nil {
		return SavingsGoal{}, mapDBError(err)
	}
	return toGoal(row), nil
}

func (s *service) DeleteGoal(ctx context.Context, a Actor, id uuid.UUID) error {
	n, err := s.repo.DeleteGoal(ctx, repo.DeleteGoalParams{ID: id, FamilyID: a.FamilyID, UserID: a.UserID})
	return affected(n, err)
}

// TopUpGoal — атомарный инкремент на сервере, а не перезапись суммы.
func (s *service) TopUpGoal(ctx context.Context, a Actor, id uuid.UUID, amount int64) (SavingsGoal, error) {
	if amount <= 0 {
		return SavingsGoal{}, ErrInvalidInput
	}
	row, err := s.repo.TopUpGoal(ctx, repo.TopUpGoalParams{Amount: amount, ID: id, FamilyID: a.FamilyID, UserID: a.UserID})
	if err != nil {
		return SavingsGoal{}, mapDBError(err)
	}
	return toGoal(row), nil
}

func validateGoal(in GoalInput) (repo.CreateGoalParams, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || (in.Target != nil && *in.Target <= 0) || in.Saved < 0 || !validScope(in.Scope) {
		return repo.CreateGoalParams{}, ErrInvalidInput
	}
	var deadline *time.Time
	if in.Deadline != nil && *in.Deadline != "" {
		d, err := parseDate(*in.Deadline)
		if err != nil {
			return repo.CreateGoalParams{}, ErrInvalidInput
		}
		deadline = &d
	}
	return repo.CreateGoalParams{Name: name, Target: in.Target, Saved: in.Saved, Deadline: deadline, Scope: in.Scope}, nil
}

// --- Общие помощники ---

func (s *service) checkCategory(ctx context.Context, a Actor, id, kind string) error {
	c, err := s.repo.GetCategory(ctx, repo.GetCategoryParams{ID: id, FamilyID: pgtype.UUID{Bytes: a.FamilyID, Valid: true}})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidInput
	}
	if err != nil {
		return err
	}
	if kind != "" && c.Kind != kind {
		return ErrInvalidInput
	}
	return nil
}

func mapDBError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrLimitExists
		case "23503", "23514":
			return ErrInvalidInput
		}
	}
	return err
}

func affected(n int64, err error) error {
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func validScope(s string) bool { return oneOf(s, "personal", "family") }

func oneOf(v string, options ...string) bool {
	for _, o := range options {
		if v == o {
			return true
		}
	}
	return false
}

func parseDate(s string) (time.Time, error) { return time.Parse(dateLayout, s) }

func today() time.Time {
	y, m, d := time.Now().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// addMonths сдвигает дату на months месяцев, обрезая день по длине целевого месяца.
func addMonths(t time.Time, months int) time.Time {
	total := t.Year()*12 + int(t.Month()) - 1 + months
	year, month := total/12, time.Month(total%12+1)
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(year, month, min(t.Day(), last), 0, 0, 0, 0, time.UTC)
}
