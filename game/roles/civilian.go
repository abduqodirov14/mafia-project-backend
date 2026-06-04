package roles

type Civilian struct{}

func (c *Civilian) Name() RoleName    { return RoleCivilian }
func (c *Civilian) Description() string { return "😇 Mafiyachilarni toping va chiqarib yuboring!" }
func (c *Civilian) IsMafia() bool     { return false }
func (c *Civilian) NightAction() bool { return false }
