package phases

import (
	"mafia-bot/game/roles"
)

const NightDuration = 60

type NightResult struct {
	KilledID  int64
	KilledName string
	Saved     bool
}

func ProcessNight(killID, saveID int64, players map[int64]string) *NightResult {
	result := &NightResult{}

	if killID == 0 {
		return result
	}

	result.KilledID = killID
	result.KilledName = players[killID]

	if killID == saveID {
		result.Saved = true
		result.KilledID = 0
	}

	return result
}

func GetMafiaVoteWinner(votes map[int64]int64) int64 {
	counts := make(map[int64]int)
	for _, target := range votes {
		counts[target]++
	}

	var winner int64
	maxVotes := 0
	for target, count := range counts {
		if count > maxVotes {
			maxVotes = count
			winner = target
		}
	}
	return winner
}

func CheckWinCondition(players map[int64]string, playerRoles map[int64]roles.RoleName) (bool, string) {
	mafiaCount := 0
	civCount := 0

	for id := range players {
		if playerRoles[id] == roles.RoleMafia {
			mafiaCount++
		} else {
			civCount++
		}
	}

	if mafiaCount == 0 {
		return true, "civilian"
	}
	if mafiaCount >= civCount {
		return true, "mafia"
	}
	return false, ""
}
