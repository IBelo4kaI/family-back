-- family_id NULL — системная категория, общая для всех семей
CREATE TABLE categories (
  id        TEXT PRIMARY KEY,
  family_id UUID REFERENCES families (id) ON DELETE CASCADE,
  name      TEXT NOT NULL,
  -- income / expense
  kind      TEXT NOT NULL CHECK (kind IN ('income', 'expense'))
);

INSERT INTO categories (id, name, kind) VALUES
  ('salary', 'Зарплата', 'income'),
  ('products', 'Продукты', 'expense'),
  ('utilities', 'Коммуналка', 'expense'),
  ('transport', 'Транспорт', 'expense'),
  ('entertainment', 'Развлечения', 'expense'),
  ('building', 'Стройматериалы', 'expense'),
  ('tools', 'Инструменты', 'expense'),
  ('payments', 'Платежи и кредиты', 'expense');

-- Во всех таблицах ниже суммы в копейках; scope = personal виден только автору (user_id / author_id)
CREATE TABLE transactions (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id   UUID NOT NULL REFERENCES families (id) ON DELETE CASCADE,
  author_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  amount      BIGINT NOT NULL CHECK (amount > 0),
  date        DATE NOT NULL,
  category_id TEXT NOT NULL REFERENCES categories (id),
  type        TEXT NOT NULL CHECK (type IN ('income', 'expense')),
  scope       TEXT NOT NULL CHECK (scope IN ('personal', 'family')),
  -- manual / receipt
  source      TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'receipt')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_transactions_family_date ON transactions (family_id, date);

CREATE TABLE recurring_payments (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id          UUID NOT NULL REFERENCES families (id) ON DELETE CASCADE,
  user_id            UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  name               TEXT NOT NULL,
  kind               TEXT NOT NULL CHECK (kind IN ('subscription', 'loan')),
  amount             BIGINT NOT NULL CHECK (amount > 0),
  period             TEXT NOT NULL CHECK (period IN ('month', 'year')),
  next_date          DATE NOT NULL,
  category_id        TEXT NOT NULL REFERENCES categories (id),
  notify_days_before INT NOT NULL CHECK (notify_days_before >= 0),
  scope              TEXT NOT NULL CHECK (scope IN ('personal', 'family')),
  status             TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'closed')),
  -- только для кредита, справочно
  total_amount       BIGINT CHECK (total_amount > 0),
  end_date           DATE,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_recurring_payments_family ON recurring_payments (family_id);

CREATE TABLE credit_cards (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id          UUID NOT NULL REFERENCES families (id) ON DELETE CASCADE,
  user_id            UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  bank               TEXT NOT NULL,
  amount_due         BIGINT NOT NULL CHECK (amount_due >= 0),
  due_date           DATE NOT NULL,
  category_id        TEXT NOT NULL REFERENCES categories (id),
  notify_days_before INT NOT NULL CHECK (notify_days_before >= 0),
  scope              TEXT NOT NULL CHECK (scope IN ('personal', 'family')),
  status             TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'closed')),
  credit_limit       BIGINT CHECK (credit_limit > 0),
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_credit_cards_family ON credit_cards (family_id);

CREATE TABLE category_limits (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id    UUID NOT NULL REFERENCES families (id) ON DELETE CASCADE,
  user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  category_id  TEXT NOT NULL REFERENCES categories (id),
  limit_amount BIGINT NOT NULL CHECK (limit_amount > 0),
  scope        TEXT NOT NULL CHECK (scope IN ('personal', 'family'))
);
-- Один лимит на категорию: в семейном режиме на семью, в личном на пользователя
CREATE UNIQUE INDEX uq_category_limits ON category_limits (
  family_id, category_id, scope,
  (CASE WHEN scope = 'personal' THEN user_id ELSE '00000000-0000-0000-0000-000000000000'::uuid END)
);

CREATE TABLE savings_goals (
  id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  family_id UUID NOT NULL REFERENCES families (id) ON DELETE CASCADE,
  user_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  name      TEXT NOT NULL,
  target    BIGINT NOT NULL CHECK (target > 0),
  saved     BIGINT NOT NULL DEFAULT 0 CHECK (saved >= 0),
  deadline  DATE NOT NULL,
  scope     TEXT NOT NULL CHECK (scope IN ('personal', 'family'))
);
CREATE INDEX idx_savings_goals_family ON savings_goals (family_id);
