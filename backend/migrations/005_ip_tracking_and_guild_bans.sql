CREATE TABLE IF NOT EXISTS user_ip_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    first_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    access_count INTEGER DEFAULT 1,
    isp_info TEXT,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, ip_address)
);

CREATE INDEX IF NOT EXISTS idx_user_ip_log_user_id ON user_ip_log(user_id);
CREATE INDEX IF NOT EXISTS idx_user_ip_log_ip_address ON user_ip_log(ip_address);
CREATE INDEX IF NOT EXISTS idx_user_ip_log_last_seen ON user_ip_log(last_seen DESC);

CREATE TABLE IF NOT EXISTS banned_guilds (
    guild_id UUID PRIMARY KEY,
    banned_by UUID NOT NULL,
    reason TEXT NOT NULL,
    banned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (guild_id) REFERENCES guilds(id) ON DELETE CASCADE,
    FOREIGN KEY (banned_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_banned_guilds_banned_at ON banned_guilds(banned_at DESC);


