CREATE TABLE families (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  name          TEXT NOT NULL,
  -- "H S% L%", без обёртки hsl()
  color         TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Одна семья на пользователя держится на уровне логики, не схемы
CREATE TABLE family_members (
  user_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  family_id UUID NOT NULL REFERENCES families (id) ON DELETE CASCADE,
  -- owner / member
  role      TEXT NOT NULL,
  joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, family_id)
);
CREATE INDEX idx_family_members_family ON family_members (family_id);

CREATE TABLE invites (
  code       TEXT PRIMARY KEY,
  family_id  UUID NOT NULL REFERENCES families (id) ON DELETE CASCADE,
  created_by UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at    TIMESTAMPTZ
);

CREATE TABLE refresh_tokens (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens (user_id);
