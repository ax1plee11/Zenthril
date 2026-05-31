CREATE TABLE IF NOT EXISTS federation_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  message_id TEXT NOT NULL UNIQUE,
  source_domain TEXT NOT NULL,
  target_domain TEXT NOT NULL,
  sender_user_id TEXT NOT NULL,
  target_user_id TEXT NOT NULL,
  payload JSONB NOT NULL,
  received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  delivered_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_federation_messages_target_received
  ON federation_messages(target_domain, received_at DESC);

CREATE INDEX IF NOT EXISTS idx_federation_messages_target_user
  ON federation_messages(target_user_id, received_at DESC);
