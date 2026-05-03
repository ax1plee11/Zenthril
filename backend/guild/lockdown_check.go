package guild

type GuildStateChecker interface {
	CanUserLeaveGuild(guildID, userID string) (bool, string)
}

func (s *Service) SetStateChecker(checker GuildStateChecker) {
	s.stateChecker = checker
}
