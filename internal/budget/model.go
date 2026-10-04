package budget

import (
	"time"

	repo "family/internal/adapter/postgres/sqlc"

	"github.com/google/uuid"
)

const dateLayout = "2006-01-02"

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type Transaction struct {
	ID         uuid.UUID `json:"id"`
	Amount     int64     `json:"amount"`
	Date       string    `json:"date"`
	CategoryID string    `json:"categoryId"`
	Type       string    `json:"type"`
	Scope      string    `json:"scope"`
	AuthorID   uuid.UUID `json:"authorId"`
	Source     string    `json:"source"`
}

type TransactionInput struct {
	Amount     int64  `json:"amount"`
	Date       string `json:"date"`
	CategoryID string `json:"categoryId"`
	Type       string `json:"type"`
	Scope      string `json:"scope"`
}

type RecurringPayment struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Kind             string    `json:"kind"`
	Amount           int64     `json:"amount"`
	Period           string    `json:"period"`
	NextDate         string    `json:"nextDate"`
	CategoryID       string    `json:"categoryId"`
	NotifyDaysBefore int32     `json:"notifyDaysBefore"`
	Scope            string    `json:"scope"`
	Status           string    `json:"status"`
	TotalAmount      *int64    `json:"totalAmount,omitempty"`
	EndDate          *string   `json:"endDate,omitempty"`
}

type PaymentInput struct {
	Name             string  `json:"name"`
	Kind             string  `json:"kind"`
	Amount           int64   `json:"amount"`
	Period           string  `json:"period"`
	NextDate         string  `json:"nextDate"`
	CategoryID       string  `json:"categoryId"`
	NotifyDaysBefore int32   `json:"notifyDaysBefore"`
	Scope            string  `json:"scope"`
	TotalAmount      *int64  `json:"totalAmount"`
	EndDate          *string `json:"endDate"`
}

type CreditCard struct {
	ID               uuid.UUID `json:"id"`
	Bank             string    `json:"bank"`
	AmountDue        int64     `json:"amountDue"`
	DueDate          string    `json:"dueDate"`
	CategoryID       string    `json:"categoryId"`
	NotifyDaysBefore int32     `json:"notifyDaysBefore"`
	Scope            string    `json:"scope"`
	Status           string    `json:"status"`
	CreditLimit      *int64    `json:"creditLimit,omitempty"`
}

type CardInput struct {
	Bank             string `json:"bank"`
	AmountDue        int64  `json:"amountDue"`
	DueDate          string `json:"dueDate"`
	CategoryID       string `json:"categoryId"`
	NotifyDaysBefore int32  `json:"notifyDaysBefore"`
	Scope            string `json:"scope"`
	CreditLimit      *int64 `json:"creditLimit"`
}

type CategoryLimit struct {
	ID         uuid.UUID `json:"id"`
	CategoryID string    `json:"categoryId"`
	Limit      int64     `json:"limit"`
	Scope      string    `json:"scope"`
}

type LimitInput struct {
	CategoryID string `json:"categoryId"`
	Limit      int64  `json:"limit"`
	Scope      string `json:"scope"`
}

type SavingsGoal struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Target   *int64    `json:"target"`
	Saved    int64     `json:"saved"`
	Deadline *string   `json:"deadline"`
	Scope    string    `json:"scope"`
}

type GoalInput struct {
	Name     string `json:"name"`
	Target   *int64  `json:"target"`
	Saved    int64   `json:"saved"`
	Deadline *string `json:"deadline"`
	Scope    string  `json:"scope"`
}

type TopUpRequest struct {
	Amount int64 `json:"amount"`
}

type SettleRequest struct {
	Action string `json:"action"` // confirm / skip
}

type SettleResult struct {
	NextDate    string       `json:"nextDate"`
	Transaction *Transaction `json:"transaction,omitempty"`
}

type Data struct {
	Categories   []Category         `json:"categories"`
	Transactions []Transaction      `json:"transactions"`
	Payments     []RecurringPayment `json:"payments"`
	Cards        []CreditCard       `json:"cards"`
	Limits       []CategoryLimit    `json:"limits"`
	Goals        []SavingsGoal      `json:"goals"`
}

func fmtDate(t time.Time) string { return t.Format(dateLayout) }

func fmtDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := fmtDate(*t)
	return &s
}

func toTransaction(t repo.Transaction) Transaction {
	return Transaction{
		ID: t.ID, Amount: t.Amount, Date: fmtDate(t.Date), CategoryID: t.CategoryID,
		Type: t.Type, Scope: t.Scope, AuthorID: t.AuthorID, Source: t.Source,
	}
}

func toPayment(p repo.RecurringPayment) RecurringPayment {
	return RecurringPayment{
		ID: p.ID, Name: p.Name, Kind: p.Kind, Amount: p.Amount, Period: p.Period,
		NextDate: fmtDate(p.NextDate), CategoryID: p.CategoryID, NotifyDaysBefore: p.NotifyDaysBefore,
		Scope: p.Scope, Status: p.Status, TotalAmount: p.TotalAmount, EndDate: fmtDatePtr(p.EndDate),
	}
}

func toCard(c repo.CreditCard) CreditCard {
	return CreditCard{
		ID: c.ID, Bank: c.Bank, AmountDue: c.AmountDue, DueDate: fmtDate(c.DueDate),
		CategoryID: c.CategoryID, NotifyDaysBefore: c.NotifyDaysBefore, Scope: c.Scope,
		Status: c.Status, CreditLimit: c.CreditLimit,
	}
}

func toLimit(l repo.CategoryLimit) CategoryLimit {
	return CategoryLimit{ID: l.ID, CategoryID: l.CategoryID, Limit: l.LimitAmount, Scope: l.Scope}
}

func toGoal(g repo.SavingsGoal) SavingsGoal {
	return SavingsGoal{
		ID: g.ID, Name: g.Name, Target: g.Target, Saved: g.Saved,
		Deadline: fmtDatePtr(g.Deadline), Scope: g.Scope,
	}
}

func mapSlice[S, D any](in []S, f func(S) D) []D {
	out := make([]D, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}
