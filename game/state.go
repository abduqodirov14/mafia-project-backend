package game

import "mafia-bot/game/roles"

type Phase string

const (
	PhaseWaiting Phase = "waiting"
	PhaseJoining Phase = "joining"
	PhaseNight   Phase = "night"
	PhaseDay     Phase = "day"
	PhaseVoting  Phase = "voting"
	PhaseEnd     Phase = "end"
)

type Player struct {
	TelegramID  int64
	Username    string
	FirstName   string
	Role        roles.RoleName
	IsAlive     bool
	JoinOrder   int
	IsBlocked   bool
	IsProtected bool
	LastWords   string
	Kills       int
	GamesPlayed int
}

type NightState struct {
	MafiaVotes        map[int64]int64
	MafiaTargetID     int64
	DoctorTargetID    int64
	KomissarTargetID  int64
	MashuqaTargetID   int64
	DaydiTargetID     int64
	TentakTargetID    int64
	ManyakTargetID    int64
	AdvokatTargetID   int64
	BodyguardTargetID int64
	KilledIDs         []int64
	SavedIDs          []int64
	BlockedIDs        []int64
	NewDonID          int64
}

type VoteState struct {
	Votes     map[int64]int64
	VoteEndAt int64
}

type GameState struct {
	RoomID     string
	ChatID     int64
	Phase      Phase
	Round      int
	Night      *NightState
	Voting     *VoteState
	KomissarID int64
}

func NewGameState(roomID string, chatID int64) *GameState {
	return &GameState{
		RoomID: roomID, ChatID: chatID,
		Phase: PhaseWaiting, Round: 0,
		Night:  newNightState(),
		Voting: &VoteState{Votes: make(map[int64]int64)},
	}
}

func newNightState() *NightState {
	return &NightState{MafiaVotes: make(map[int64]int64)}
}

func (gs *GameState) ResetNight() { gs.Night = newNightState() }
func (gs *GameState) ResetVoting() { gs.Voting = &VoteState{Votes: make(map[int64]int64)} }
