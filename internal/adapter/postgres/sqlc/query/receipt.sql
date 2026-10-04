-- name: CreateReceiptTransaction :one
INSERT INTO transactions (family_id, author_id, amount, date, category_id, type, scope, source, seller_inn, seller_name, receipt_key)
VALUES (sqlc.arg(family_id), sqlc.arg(user_id), sqlc.arg(amount), sqlc.arg(date), sqlc.arg(category_id), 'expense', sqlc.arg(scope), 'receipt', sqlc.arg(seller_inn), sqlc.arg(seller_name), sqlc.arg(receipt_key))
RETURNING *;

-- name: CreateTransactionItem :exec
INSERT INTO transaction_items (transaction_id, position, name, price, quantity, sum)
VALUES (sqlc.arg(transaction_id), sqlc.arg(position), sqlc.arg(name), sqlc.arg(price), sqlc.arg(quantity), sqlc.arg(sum));

-- name: ListTransactionItems :many
SELECT i.id, i.position, i.name, i.price, i.quantity, i.sum, i.category_id
FROM transaction_items i
JOIN transactions t ON t.id = i.transaction_id
WHERE i.transaction_id = sqlc.arg(transaction_id) AND t.family_id = sqlc.arg(family_id) AND (t.scope = 'family' OR t.author_id = sqlc.arg(user_id))
ORDER BY i.position;
