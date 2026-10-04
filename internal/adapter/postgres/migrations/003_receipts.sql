-- Данные чека у транзакции, добавленной сканированием QR
ALTER TABLE transactions
  ADD COLUMN seller_inn  TEXT,
  ADD COLUMN seller_name TEXT,
  -- fn-fd-fp: защита от повторного добавления одного и того же чека
  ADD COLUMN receipt_key TEXT;
CREATE UNIQUE INDEX uq_transactions_receipt ON transactions (family_id, receipt_key) WHERE receipt_key IS NOT NULL;

CREATE TABLE transaction_items (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id UUID NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
  position       INT NOT NULL,
  name           TEXT NOT NULL,
  -- цена и сумма в копейках
  price          BIGINT NOT NULL CHECK (price >= 0),
  quantity       DOUBLE PRECISION NOT NULL CHECK (quantity > 0),
  sum            BIGINT NOT NULL CHECK (sum >= 0),
  -- собственная категория позиции; NULL — как у транзакции
  category_id    TEXT REFERENCES categories (id)
);
CREATE INDEX idx_transaction_items_tx ON transaction_items (transaction_id);
