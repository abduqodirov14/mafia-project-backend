package roles

type Detective struct{}

func (d *Detective) Name() RoleName    { return RoleDetective }
func (d *Detective) Description() string { return "🕵️ Har kecha bir o'yinchining rolini aniqlang." }
func (d *Detective) IsMafia() bool     { return false }
func (d *Detective) NightAction() bool { return true }
