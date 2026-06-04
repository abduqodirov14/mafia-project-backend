package game

type Phase string

const (
	PhaseWaiting Phase = "waiting"
	PhaseNight   Phase = "night"
	PhaseDay     Phase = "day"
	PhaseVoting  Phase = "voting"
	PhaseConfirm Phase = "confirm"
	PhaseEnd     Phase = "end"
)

type GameState struct {
	RoomID       string
	Phase        Phase
	Round        int
	NightKillID  int64
	DoctorSaveID int64
	BodyguardID  int64
	ConfirmID    int64
	Votes        map[int64]int64
	ConfirmVotes map[int64]bool
	MafiaVotes   map[int64]int64
	DetectiveCh  int64
}

func NewGameState(roomID string) *GameState {
	return &GameState{
		RoomID:     roomID,
		Phase:      PhaseWaiting,
		Round:      0,
		Votes:      make(map[int64]int64),
		ConfirmVotes: make(map[int64]bool),
		MafiaVotes: make(map[int64]int64),
	}
}

func (gs *GameState) ResetNight() {
	gs.NightKillID = 0
	gs.DoctorSaveID = 0
	gs.BodyguardID = 0
	gs.DetectiveCh = 0
	gs.MafiaVotes = make(map[int64]int64)
}

func (gs *GameState) ResetDay() {
	gs.Votes = make(map[int64]int64)
	gs.ConfirmID = 0
	gs.ConfirmVotes = make(map[int64]bool)
}
