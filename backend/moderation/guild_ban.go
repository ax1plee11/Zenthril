package moderation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GuildModerator struct {
	db      *pgxpool.Pool
	monitor *ConnectionMonitor
}

func NewGuildModerator(db *pgxpool.Pool, monitor *ConnectionMonitor) *GuildModerator {
	return &GuildModerator{
		db:      db,
		monitor: monitor,
	}
}

func (m *GuildModerator) ProcessGuildAction(ctx context.Context, guildID, adminID, reason string) (*GuildBanReport, error) {
	members, err := m.getGuildMembers(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("get guild members: %w", err)
	}

	membersIPs, err := m.monitor.GetGuildMembersIPs(ctx, guildID)
	if err != nil {
		slog.Warn("security moderation failed to get member data", "guild_id", guildID, "error", err)
		membersIPs = make(map[string][]ConnectionRecord)
	}

	err = m.deleteGuild(ctx, guildID, adminID, reason)
	if err != nil {
		return nil, fmt.Errorf("ban guild: %w", err)
	}

	report := &GuildBanReport{
		GuildID:    guildID,
		Reason:     reason,
		BannedBy:   adminID,
		MemberIPs:  membersIPs,
		TotalUsers: len(members),
	}

	slog.Info("security moderation guild processed", "guild_id", guildID, "admin_id", adminID, "member_count", len(members))

	return report, nil
}

func (m *GuildModerator) ProcessCascadeAction(ctx context.Context, guildID, adminID, reason string) (*CascadeBanReport, error) {
	members, err := m.getGuildMembers(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("get guild members: %w", err)
	}

	membersIPs, err := m.monitor.GetGuildMembersIPs(ctx, guildID)
	if err != nil {
		slog.Warn("security moderation failed to get member data", "guild_id", guildID, "error", err)
		membersIPs = make(map[string][]ConnectionRecord)
	}

	err = m.deleteGuild(ctx, guildID, adminID, reason)
	if err != nil {
		return nil, fmt.Errorf("delete guild: %w", err)
	}

	report := &CascadeBanReport{
		GuildID:      guildID,
		Reason:       reason,
		DeletedBy:    adminID,
		TotalMembers: len(members),
		MemberIPs:    membersIPs,
	}

	slog.Info("security moderation guild deleted", "guild_id", guildID, "admin_id", adminID, "member_count", len(members))

	return report, nil
}

func (m *GuildModerator) getGuildMembers(ctx context.Context, guildID string) ([]string, error) {
	query := `
		SELECT user_id 
		FROM guild_members 
		WHERE guild_id = $1
	`

	rows, err := m.db.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		members = append(members, userID)
	}

	return members, nil
}

func (m *GuildModerator) deleteGuild(ctx context.Context, guildID, adminID, reason string) error {
	query := `
		INSERT INTO banned_guilds (guild_id, banned_by, reason, banned_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (guild_id) DO UPDATE SET
			banned_by = EXCLUDED.banned_by,
			reason = EXCLUDED.reason,
			banned_at = NOW()
	`
	_, err := m.db.Exec(ctx, query, guildID, adminID, reason)
	if err != nil {
		return err
	}

	_, err = m.db.Exec(ctx, `DELETE FROM guilds WHERE id = $1`, guildID)
	return err
}

type GuildBanReport struct {
	GuildID    string                        `json:"guild_id"`
	Reason     string                        `json:"reason"`
	BannedBy   string                        `json:"banned_by"`
	TotalUsers int                           `json:"total_users"`
	MemberIPs  map[string][]ConnectionRecord `json:"member_ips"`
}

type CascadeBanReport struct {
	GuildID      string                        `json:"guild_id"`
	Reason       string                        `json:"reason"`
	DeletedBy    string                        `json:"deleted_by"`
	TotalMembers int                           `json:"total_members"`
	MemberIPs    map[string][]ConnectionRecord `json:"member_ips"`
}

