-- Миграция 003: система друзей
CREATE TABLE IF NOT EXISTS friendships (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id UUID       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id UUID       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      VARCHAR(10) NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending','accepted','declined','blocked')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (requester_id, addressee_id)
);

CREATE INDEX IF NOT EXISTS idx_friendships_addressee ON friendships(addressee_id, status);
CREATE INDEX IF NOT EXISTS idx_friendships_requester ON friendships(requester_id, status);
