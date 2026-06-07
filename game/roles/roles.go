package roles

type RoleName string
type Side string

const (
	SideTown    Side = "town"
	SideMafia   Side = "mafia"
	SideNeutral Side = "neutral"
)

// ─── SHAHAR TOMONLARI ───
const (
	RoleCivilian  RoleName = "Tinch aholi"
	RoleDoctor    RoleName = "Shifokor"
	RoleKomissar  RoleName = "Komissar"
	RoleSerjant   RoleName = "Serjant"
	RoleDaydi     RoleName = "Daydi"
	RoleMashuqa   RoleName = "Mashuqa"
	RoleKamikaze  RoleName = "Kamikaze"
	RoleTentak    RoleName = "Tentak"
)

// ─── MAFIA TOMONLARI ───
const (
	RoleDon     RoleName = "Don"
	RoleMafia   RoleName = "Mafiya"
	RoleAdvokat RoleName = "Advokat"
)

// ─── YAKKA ROLLAR ───
const (
	RoleManyak  RoleName = "Manyak"
	RoleSuidsid RoleName = "Suidsid"
)

type RoleInfo struct {
	Name        RoleName
	Side        Side
	Emoji       string
	Description string
	NightAction bool
	Priority    int // Kecha harakatlar tartibi
}

var Catalog = map[RoleName]RoleInfo{
	// SHAHAR
	RoleCivilian: {RoleCivilian, SideTown, "👨🏼", "Oddiy fuqaro. Mafiyani toping va ovoz berish orqali chiqarib yuboring.", false, 0},
	RoleDoctor:   {RoleDoctor, SideTown, "👨‍⚕️", "Tunda bitta o'yinchini o'limdan qutqaradi. O'zini faqat 1 marta davolay oladi.", true, 30},
	RoleKomissar: {RoleKomissar, SideTown, "🕵🏼", "Tunda bitta o'yinchini tekshiradi. 1-tunda tekshiruv qila olmaydi.", true, 40},
	RoleSerjant:  {RoleSerjant, SideTown, "👮‍♂️", "Komissarga yordam beradi. Komissar o'lsa — uning o'rnini egallaydi.", true, 41},
	RoleDaydi:    {RoleDaydi, SideTown, "🧙🏻‍♂️", "Tunda bir o'yinchining uyida tunaydi va u yerga kelgan barcha rollarni ko'radi.", true, 10},
	RoleMashuqa:  {RoleMashuqa, SideTown, "💃", "Tunda istalgan bitta o'yinchini tanlaydi. U bilan tunni o'tkazadi — tanlangan harakat qila olmaydi.", true, 5},
	RoleKamikaze: {RoleKamikaze, SideTown, "💣", "Osilsa → o'zi bilan bitta o'yinchini olib ketadi. Otib o'ldirilsa → otgan odam ham o'ladi.", false, 0},
	RoleTentak:   {RoleTentak, SideTown, "👨🏻‍🦲", "Tinch aholini tanlasa — osishdan himoya qiladi. Mafia/yakka rollarni tanlasa — o'ldiradi.", true, 35},
	// MAFIA
	RoleDon:     {RoleDon, SideMafia, "🤵🏻", "Mafiya boshlig'i. Qotillikda oxirgi qarorni beradi. Komissar tekshirsa oddiy odamdek ko'rinadi.", true, 20},
	RoleMafia:   {RoleMafia, SideMafia, "🤵🏼", "Donning buyruqlarini bajaradi. Don o'lsa — Don bo'ladi.", true, 21},
	RoleAdvokat: {RoleAdvokat, SideMafia, "👨‍💼", "Komissar tekshirsa tinch aholi bo'lib ko'rinadi. Mafiyani yashiradi.", true, 22},
	// YAKKA
	RoleManyak:  {RoleManyak, SideNeutral, "🔪", "Hamma o'lsin — faqat u tirik qolsin. Har kecha bitta odamni o'ldiradi.", true, 25},
	RoleSuidsid: {RoleSuidsid, SideNeutral, "🧌", "Ovoz bilan chiqarib yuborilsa — barcha yutqazadi, faqat u yutadi.", false, 0},
}

func Get(name RoleName) RoleInfo {
	if info, ok := Catalog[name]; ok {
		return info
	}
	return Catalog[RoleCivilian]
}

func IsMafia(name RoleName) bool {
	return Get(name).Side == SideMafia
}

func HasNightAction(name RoleName) bool {
	return Get(name).NightAction
}

// Rol taqsimlash — n ta o'yinchi uchun
func AssignRoles(n int) []RoleName {
	assigned := make([]RoleName, n)

	// Mafia soni
	mafiaCount := n / 4
	if mafiaCount == 0 {
		mafiaCount = 1
	}

	idx := 0
	// Don (birinchi mafia)
	assigned[idx] = RoleDon
	idx++

	// Qolgan mafia
	for i := 1; i < mafiaCount; i++ {
		assigned[idx] = RoleMafia
		idx++
	}

	// Shifokor
	if idx < n {
		assigned[idx] = RoleDoctor
		idx++
	}
	// Komissar
	if idx < n {
		assigned[idx] = RoleKomissar
		idx++
	}
	// Mashuqa (6+ kishi)
	if n >= 6 && idx < n {
		assigned[idx] = RoleMashuqa
		idx++
	}
	// Daydi (7+ kishi)
	if n >= 7 && idx < n {
		assigned[idx] = RoleDaydi
		idx++
	}
	// Advokat (7+ kishi)
	if n >= 7 && idx < n {
		assigned[idx] = RoleAdvokat
		idx++
	}
	// Serjant (8+ kishi)
	if n >= 8 && idx < n {
		assigned[idx] = RoleSerjant
		idx++
	}
	// Manyak (9+ kishi)
	if n >= 9 && idx < n {
		assigned[idx] = RoleManyak
		idx++
	}
	// Kamikaze (10+ kishi)
	if n >= 10 && idx < n {
		assigned[idx] = RoleKamikaze
		idx++
	}
	// Suidsid (11+ kishi)
	if n >= 11 && idx < n {
		assigned[idx] = RoleSuidsid
		idx++
	}

	// Qolganlar — Tinch aholi
	for idx < n {
		assigned[idx] = RoleCivilian
		idx++
	}

	return assigned
}
