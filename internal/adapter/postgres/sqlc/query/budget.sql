-- name: ListCategories :many
SELECT id, name, kind FROM categories WHERE family_id IS NULL OR family_id = $1 ORDER BY family_id NULLS FIRST, name;

-- name: GetCategory :one
SELECT id, name, kind FROM categories WHERE id = $1 AND (family_id IS NULL OR family_id = $2);

-- name: ListTransactions :many
SELECT * FROM transactions WHERE family_id = sqlc.arg(family_id) AND (scope = 'family' OR author_id = sqlc.arg(user_id)) ORDER BY date DESC, created_at DESC;

-- name: CreateTransaction :one
INSERT INTO transactions (family_id, author_id, amount, date, category_id, type, scope, source)
VALUES (sqlc.arg(family_id), sqlc.arg(user_id), $1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateTransaction :one
UPDATE transactions SET amount = $1, date = $2, category_id = $3, type = $4, scope = sqlc.arg(scope)
WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR author_id = sqlc.arg(user_id)) AND (sqlc.arg(scope)::text = 'family' OR author_id = sqlc.arg(user_id))
RETURNING *;

-- name: DeleteTransaction :execrows
DELETE FROM transactions WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR author_id = sqlc.arg(user_id));

-- name: ListPayments :many
SELECT * FROM recurring_payments WHERE family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) ORDER BY created_at;

-- name: GetPaymentForUpdate :one
SELECT * FROM recurring_payments WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) FOR UPDATE;

-- name: CreatePayment :one
INSERT INTO recurring_payments (family_id, user_id, name, kind, amount, period, next_date, category_id, notify_days_before, scope, total_amount, end_date)
VALUES (sqlc.arg(family_id), sqlc.arg(user_id), sqlc.arg(name), sqlc.arg(kind), sqlc.arg(amount), sqlc.arg(period), sqlc.arg(next_date), sqlc.arg(category_id), sqlc.arg(notify_days_before), sqlc.arg(scope), sqlc.narg(total_amount), sqlc.narg(end_date))
RETURNING *;

-- name: UpdatePayment :one
UPDATE recurring_payments SET name = sqlc.arg(name), kind = sqlc.arg(kind), amount = sqlc.arg(amount), period = sqlc.arg(period),
  next_date = sqlc.arg(next_date), category_id = sqlc.arg(category_id), notify_days_before = sqlc.arg(notify_days_before),
  scope = sqlc.arg(scope), total_amount = sqlc.narg(total_amount), end_date = sqlc.narg(end_date)
WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) AND (sqlc.arg(scope)::text = 'family' OR user_id = sqlc.arg(user_id))
RETURNING *;

-- name: ClosePayment :execrows
UPDATE recurring_payments SET status = 'closed' WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id));

-- name: SetPaymentNextDate :exec
UPDATE recurring_payments SET next_date = $2 WHERE id = $1;

-- name: ListCards :many
SELECT * FROM credit_cards WHERE family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) ORDER BY created_at;

-- name: GetCardForUpdate :one
SELECT * FROM credit_cards WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) FOR UPDATE;

-- name: CreateCard :one
INSERT INTO credit_cards (family_id, user_id, bank, amount_due, due_date, category_id, notify_days_before, scope, credit_limit)
VALUES (sqlc.arg(family_id), sqlc.arg(user_id), sqlc.arg(bank), sqlc.arg(amount_due), sqlc.arg(due_date), sqlc.arg(category_id), sqlc.arg(notify_days_before), sqlc.arg(scope), sqlc.narg(credit_limit))
RETURNING *;

-- name: UpdateCard :one
UPDATE credit_cards SET bank = sqlc.arg(bank), amount_due = sqlc.arg(amount_due), due_date = sqlc.arg(due_date),
  category_id = sqlc.arg(category_id), notify_days_before = sqlc.arg(notify_days_before), scope = sqlc.arg(scope),
  credit_limit = sqlc.narg(credit_limit)
WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) AND (sqlc.arg(scope)::text = 'family' OR user_id = sqlc.arg(user_id))
RETURNING *;

-- name: CloseCard :execrows
UPDATE credit_cards SET status = 'closed' WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id));

-- name: SetCardDueDate :exec
UPDATE credit_cards SET due_date = $2 WHERE id = $1;

-- name: ListLimits :many
SELECT * FROM category_limits WHERE family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) ORDER BY category_id;

-- name: CreateLimit :one
INSERT INTO category_limits (family_id, user_id, category_id, limit_amount, scope)
VALUES (sqlc.arg(family_id), sqlc.arg(user_id), $1, $2, $3)
RETURNING *;

-- name: UpdateLimit :one
UPDATE category_limits SET category_id = $1, limit_amount = $2, scope = sqlc.arg(scope)
WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) AND (sqlc.arg(scope)::text = 'family' OR user_id = sqlc.arg(user_id))
RETURNING *;

-- name: DeleteLimit :execrows
DELETE FROM category_limits WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id));

-- name: ListGoals :many
SELECT * FROM savings_goals WHERE family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) ORDER BY deadline;

-- name: CreateGoal :one
INSERT INTO savings_goals (family_id, user_id, name, target, saved, deadline, scope)
VALUES (sqlc.arg(family_id), sqlc.arg(user_id), $1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateGoal :one
UPDATE savings_goals SET name = $1, target = $2, saved = $3, deadline = $4, scope = sqlc.arg(scope)
WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id)) AND (sqlc.arg(scope)::text = 'family' OR user_id = sqlc.arg(user_id))
RETURNING *;

-- name: DeleteGoal :execrows
DELETE FROM savings_goals WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id));

-- name: TopUpGoal :one
UPDATE savings_goals SET saved = saved + sqlc.arg(amount)
WHERE id = sqlc.arg(id) AND family_id = sqlc.arg(family_id) AND (scope = 'family' OR user_id = sqlc.arg(user_id))
RETURNING *;
