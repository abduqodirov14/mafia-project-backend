package phases

func GetVoteWinner(votes map[int64]int64) (int64, bool) {
	counts := make(map[int64]int)
	for _, target := range votes {
		counts[target]++
	}

	var winner int64
	maxVotes := 0
	tie := false

	for target, count := range counts {
		if count > maxVotes {
			maxVotes = count
			winner = target
			tie = false
		} else if count == maxVotes {
			tie = true
		}
	}

	if tie {
		return 0, true
	}
	return winner, false
}
