package guild

import "fmt"

type GuildStateChecker interface {
	CanUserLeaveGuild(guildID, userID string) (bool, string)
}

var ErrGuildLocked = fmt.Errorf("guild_locked")

func (s *Service) SetStateChecker(checker GuildStateChecker) {
	s.stateChecker = checker
}
