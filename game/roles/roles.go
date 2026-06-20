package roles

type RoleName string
type Side string

const (
	SideTown    Side = "town"
	SideMafia   Side = "mafia"
	SideNeutral Side = "neutral"
)

const (
	RoleCivilian  RoleName = "Tinch aholi"
	RoleDoctor    RoleName = "Shifokor"
	RoleKomissar  RoleName = "Komissar"
	RoleSerjant   RoleName = "Serjant"
	RoleDaydi     RoleName = "Daydi"
	RoleMashuqa   RoleName = "Mashuqa"
	RoleKamikaze  RoleName = "Kamikaze"
	RoleTentak    RoleName = "Tentak"
	RoleDon       RoleName = "Don"
	RoleMafia     RoleName = "Mafiya"
	RoleAdvokat   RoleName = "Advokat"
	RoleManyak    RoleName = "Manyak"
	RoleSuidsid   RoleName = "Suidsid"
)

type Info struct {
	Name        RoleName
	Side        Side
	Emoji       string
	Description string
	NightAction bool
	Priority    int
}

var catalog = map[RoleName]Info{
	RoleCivilian: {RoleCivilian, SideTown, "👨🏼", "Oddiy fuqaro. Mafiyani toping va ovoz berish orqali chiqarib yuboring.", false, 0},
	RoleDoctor:   {RoleDoctor, SideTown, "👨‍⚕️", "Tunda bitta o'yinchini o'limdan qutqaradi. O'zini faqat 1 marta davolay oladi.", true, 30},
	RoleKomissar: {RoleKomissar, SideTown, "🕵🏼", "Tunda bitta o'yinchini tekshiradi. 1-tunda tekshiruv qila olmaydi.", true, 40},
	RoleSerjant:  {RoleSerjant, SideTown, "👮‍♂️", "Komissarga yordam beradi. Komissar o'lsa — uning o'rnini egallaydi.", true, 41},
	RoleDaydi:    {RoleDaydi, SideTown, "🧙🏻‍♂️", "Tunda bir o'yinchining uyida tunaydi va u yerga kelgan barcha rollarni ko'radi.", true, 10},
	RoleMashuqa:  {RoleMashuqa, SideTown, "💃", "Tunda istalgan bitta o'yinchini tanlaydi. U bilan tunni o'tkazadi — tanlangan harakat qila olmaydi.", true, 5},
	RoleKamikaze: {RoleKamikaze, SideTown, "💣", "Osilsa → o'zi bilan bitta o'yinchini olib ketadi. Otib o'ldirilsa → otgan odam ham o'ladi.", false, 0},
	RoleTentak:   {RoleTentak, SideTown, "👨🏻‍🦲", "Tinch aholini tanlasa — osishdan himoya qiladi. Mafia/yakka rollarni tanlasa — o'ldiradi.", true, 35},
	RoleDon:      {RoleDon, SideMafia, "🤵🏻", "Mafiya boshlig'i. Qotillikda oxirgi qarorni beradi. Komissar tekshirsa oddiy odamdek ko'rinadi.", true, 20},
	RoleMafia:    {RoleMafia, SideMafia, "🤵🏼", "Donning buyruqlarini bajaradi. Don o'lsa — Don bo'ladi.", true, 21},
	RoleAdvokat:  {RoleAdvokat, SideMafia, "👨‍💼", "Komissar tekshirsa tinch aholi bo'lib ko'rinadi. Mafiyani yashiradi.", true, 22},
	RoleManyak:   {RoleManyak, SideNeutral, "🔪", "Hamma o'lsin — faqat u tirik qolsin. Har kecha bitta odamni o'ldiradi.", true, 25},
	RoleSuidsid:  {RoleSuidsid, SideNeutral, "🧌", "Ovoz bilan chiqarib yuborilsa — barcha yutqazadi, faqat u yutadi.", false, 0},
}

func Get(name RoleName) Info {
	if info, ok := catalog[name]; ok {
		return info
	}
	return catalog[RoleCivilian]
}

func IsMafia(name RoleName) bool {
	return Get(name).Side == SideMafia
}

func HasNightAction(name RoleName) bool {
	return Get(name).NightAction
}

func AssignRoles(n int) []RoleName {
	if n < 1 {
		return nil
	}

	assigned := make([]RoleName, 0, n)

	// Mafia soni: har 4 kishiga 1 ta, lekin kamida 1 ta
	mafiaCount := n / 4
	if mafiaCount < 1 {
		mafiaCount = 1
	}

	// Mafia: Don + qolganlari
	assigned = append(assigned, RoleDon)
	for i := 1; i < mafiaCount; i++ {
		assigned = append(assigned, RoleMafia)
	}

	// Shahar rollari (majburiy)
	assigned = appendIfRoom(assigned, n, RoleDoctor)
	assigned = appendIfRoom(assigned, n, RoleKomissar)

	// Shartli rollar (o'yinchi soniga qarab)
	type conditionalRole struct {
		minPlayers int
		role       RoleName
	}
	conditionalRoles := []conditionalRole{
		{6, RoleMashuqa},
		{7, RoleDaydi},
		{7, RoleAdvokat},
		{8, RoleSerjant},
		{9, RoleManyak},
		{10, RoleKamikaze},
		{11, RoleSuidsid},
	}

	for _, cr := range conditionalRoles {
		if n >= cr.minPlayers {
			assigned = appendIfRoom(assigned, n, cr.role)
		}
	}

	// Qolganlari — Tinch aholi
	for len(assigned) < n {
		assigned = append(assigned, RoleCivilian)
	}

	return assigned
}

func appendIfRoom(roles []RoleName, maxLen int, role RoleName) []RoleName {
	if len(roles) < maxLen {
		return append(roles, role)
	}
	return roles
}
