package roles

type Doctor struct{}

func (d *Doctor) Name() RoleName    { return RoleDoctor }
func (d *Doctor) Description() string { return "👨‍⚕️ Har kecha bir o'yinchini davolang." }
func (d *Doctor) IsMafia() bool     { return false }
func (d *Doctor) NightAction() bool { return true }
