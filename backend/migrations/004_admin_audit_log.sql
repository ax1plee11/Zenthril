-- Таблица для аудита действий администраторов
CREATE TABLE IF NOT EXISTS admin_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    target_id UUID,
    details TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (admin_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_admin_audit_admin_id ON admin_audit_log(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_created_at ON admin_audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_action ON admin_audit_log(action);
CREATE INDEX IF NOT EXISTS idx_admin_audit_target_id ON admin_audit_log(target_id);

-- Комментарии для документации
COMMENT ON TABLE admin_audit_log IS 'Аудит лог всех действий администраторов для безопасности';
COMMENT ON COLUMN admin_audit_log.admin_id IS 'UUID администратора, выполнившего действие';
COMMENT ON COLUMN admin_audit_log.action IS 'Тип действия (GLOBAL_BAN, GLOBAL_UNBAN, и т.д.)';
COMMENT ON COLUMN admin_audit_log.target_id IS 'UUID пользователя, над которым выполнено действие';
COMMENT ON COLUMN admin_audit_log.details IS 'Дополнительные детали (причина бана и т.д.)';
COMMENT ON COLUMN admin_audit_log.ip_address IS 'IP адрес администратора';
