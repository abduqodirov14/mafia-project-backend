package roles

type Mafia struct{}

func (m *Mafia) Name() RoleName    { return RoleMafia }
func (m *Mafia) Description() string { return "😈 Har kecha bir tinch aholini o'ldiring." }
func (m *Mafia) IsMafia() bool     { return true }
func (m *Mafia) NightAction() bool { return true }
