package guild

import (
	"errors"
)

type GuildStateChecker interface {
	CanUserLeaveGuild(guildID, userID string) (bool, string)
	OnOwnerJoinGuild(guildID, userID string)
	OnOwnerLeaveGuild(guildID, userID string)
}

var (
	ErrGuildLocked = errors.New("guild_locked")
)

func (s *Service) SetStateChecker(checker GuildStateChecker) {
	s.stateChecker = checker
}
