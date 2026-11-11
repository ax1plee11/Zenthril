package moderation

import (
	"context"
	"log"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnectionMonitor struct {
	db *pgxpool.Pool
}

func NewConnectionMonitor(db *pgxpool.Pool) *ConnectionMonitor {
	return &ConnectionMonitor{db: db}
}

func ExtractRealIP(headers map[string]string, remoteAddr string) []string {
	var ips []string

	possibleHeaders := []string{
		"CF-Connecting-IP",
		"True-Client-IP",
		"X-Real-IP",
		"X-Forwarded-For",
		"X-Client-IP",
		"X-Cluster-Client-IP",
		"Forwarded-For",
		"Forwarded",
	}

	for _, header := range possibleHeaders {
		if value, ok := headers[header]; ok && value != "" {
			parts := strings.Split(value, ",")
			for _, part := range parts {
				ip := strings.TrimSpace(part)
				if isValidIP(ip) {
					ips = append(ips, ip)
				}
			}
		}
	}

	if remoteAddr != "" {
		host, _, err := net.SplitHostPort(remoteAddr)
		if err == nil && isValidIP(host) {
			ips = append(ips, host)
		} else if isValidIP(remoteAddr) {
			ips = append(ips, remoteAddr)
		}
	}

	return ips
}

func isValidIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	if parsed.IsLoopback() || parsed.IsPrivate() {
		return false
	}
	return true
}

func (t *ConnectionMonitor) RecordConnection(ctx context.Context, userID string, ips []string, userAgent string) error {
	if len(ips) == 0 {
		return nil
	}

	for _, ip := range ips {
		query := `
			INSERT INTO user_ip_log (user_id, ip_address, user_agent, first_seen, last_seen, access_count)
			VALUES ($1, $2, $3, NOW(), NOW(), 1)
			ON CONFLICT (user_id, ip_address) 
			DO UPDATE SET 
				last_seen = NOW(),
				access_count = user_ip_log.access_count + 1,
				user_agent = EXCLUDED.user_agent
		`

		_, err := t.db.Exec(ctx, query, userID, ip, userAgent)
		if err != nil {
			log.Printf("[SERVER] Connection record error for user %s: %v", userID, err)
		}
	}

	return nil
}

func (t *ConnectionMonitor) GetUserIPs(ctx context.Context, userID string) ([]ConnectionRecord, error) {
	query := `
		SELECT ip_address, user_agent, first_seen, last_seen, access_count, isp_info
		FROM user_ip_log
		WHERE user_id = $1
		ORDER BY last_seen DESC
	`

	rows, err := t.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ConnectionRecord
	for rows.Next() {
		var r ConnectionRecord
		err := rows.Scan(&r.IP, &r.UserAgent, &r.FirstSeen, &r.LastSeen, &r.AccessCount, &r.ISPInfo)
		if err != nil {
			continue
		}
		records = append(records, r)
	}

	return records, nil
}

func (t *ConnectionMonitor) GetGuildMembersIPs(ctx context.Context, guildID string) (map[string][]ConnectionRecord, error) {
	query := `
		SELECT DISTINCT 
			gm.user_id,
			uil.ip_address,
			uil.user_agent,
			uil.first_seen,
			uil.last_seen,
			uil.access_count,
			uil.isp_info
		FROM guild_members gm
		JOIN user_ip_log uil ON gm.user_id = uil.user_id
		WHERE gm.guild_id = $1
		ORDER BY gm.user_id, uil.last_seen DESC
	`

	rows, err := t.db.Query(ctx, query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]ConnectionRecord)
	for rows.Next() {
		var userID string
		var r ConnectionRecord
		err := rows.Scan(&userID, &r.IP, &r.UserAgent, &r.FirstSeen, &r.LastSeen, &r.AccessCount, &r.ISPInfo)
		if err != nil {
			continue
		}
		result[userID] = append(result[userID], r)
	}

	return result, nil
}

func (t *ConnectionMonitor) EnrichIPInfo(ctx context.Context, ip string) (*NetworkInfo, error) {
	info := &NetworkInfo{IP: ip}

	parsed := net.ParseIP(ip)
	if parsed != nil {
		if parsed.To4() != nil {
			info.Type = "IPv4"
		} else {
			info.Type = "IPv6"
		}
	}

	query := `
		UPDATE user_ip_log 
		SET isp_info = $2
		WHERE ip_address = $1
	`

	_, err := t.db.Exec(ctx, query, ip, info.Type)

	return info, err
}

type ConnectionRecord struct {
	IP          string    `json:"ip"`
	UserAgent   string    `json:"user_agent"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	AccessCount int       `json:"access_count"`
	ISPInfo     string    `json:"isp_info"`
}

type NetworkInfo struct {
	IP      string `json:"ip"`
	Type    string `json:"type"`
	Country string `json:"country"`
	City    string `json:"city"`
	ISP     string `json:"isp"`
	IsVPN   bool   `json:"is_vpn"`
	IsProxy bool   `json:"is_proxy"`
	IsTor   bool   `json:"is_tor"`
}
