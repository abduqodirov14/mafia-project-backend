package roles

type RoleName string

const (
	RoleMafia        RoleName = "Mafia"
	RoleDon          RoleName = "Don"
	RoleGodfather    RoleName = "Godfather"
	RoleCivilian     RoleName = "Tinch aholi"
	RoleDoctor       RoleName = "Doktor"
	RoleDetective    RoleName = "Detektiv"
	RoleSheriff      RoleName = "Sheriff"
	RoleBodyguard    RoleName = "Bodyguard"
	RoleVigilante    RoleName = "Vigilante"
	RoleJudge        RoleName = "Judge"
	RolePriest       RoleName = "Priest"
	RoleSpy          RoleName = "Spy"
	RoleNurse        RoleName = "Nurse"
	RoleMedium       RoleName = "Medium"
	RoleTracker      RoleName = "Tracker"
	RoleWatcher      RoleName = "Watcher"
	RoleSilencer     RoleName = "Silencer"
	RoleFramer       RoleName = "Framer"
	RoleAssassin     RoleName = "Assassin"
	RoleBomber       RoleName = "Bomber"
	RolePoisoner     RoleName = "Poisoner"
	RoleNinja        RoleName = "Ninja"
	RoleWerewolf     RoleName = "Werewolf"
	RoleJester       RoleName = "Jester"
	RoleSerialKiller RoleName = "Serial Killer"
	RoleVampire      RoleName = "Vampire"
	RoleArsonist     RoleName = "Arsonist"
	RoleJoker        RoleName = "Joker"
	RoleCultLeader   RoleName = "Cult Leader"
	RoleLover        RoleName = "Lover"
	RoleCupid        RoleName = "Cupid"
	RoleTwin         RoleName = "Twin"
)

type Role interface {
	Name() RoleName
	Description() string
	IsMafia() bool
	NightAction() bool
}

type Side string

const (
	SideTown    Side = "town"
	SideMafia   Side = "mafia"
	SideNeutral Side = "neutral"
	SideSpecial Side = "special"
)

type RoleInfo struct {
	Name        RoleName
	Side        Side
	Emoji       string
	Description string
	Timing      string
	NightAction bool
}

var Catalog = []RoleInfo{
	{Name: RoleCivilian, Side: SideTown, Emoji: "Citizen", Description: "Oddiy fuqaro. Maxsus kuchi yo'q, kunduzgi muhokama va ovoz berishda mafiyani topishga yordam beradi.", Timing: "day"},
	{Name: RoleSheriff, Side: SideTown, Emoji: "Sheriff", Description: "Har tunda bitta o'yinchini tekshiradi va uning mafia yoki oddiy ekanini biladi.", Timing: "night", NightAction: true},
	{Name: RoleDetective, Side: SideTown, Emoji: "Detective", Description: "O'yinchilar haqida ma'lumot yig'adi. Ba'zi qoidalarda rolni yoki gumon darajasini ko'radi.", Timing: "night", NightAction: true},
	{Name: RoleDoctor, Side: SideTown, Emoji: "Doctor", Description: "Har tunda bitta o'yinchini davolaydi. Agar unga hujum bo'lsa, tirik saqlashi mumkin.", Timing: "night", NightAction: true},
	{Name: RoleBodyguard, Side: SideTown, Emoji: "Shield", Description: "Bitta o'yinchini himoya qiladi. Hujum bo'lsa, himoyalangan odam o'rniga zarba olishi mumkin.", Timing: "night", NightAction: true},
	{Name: RoleVigilante, Side: SideTown, Emoji: "Gun", Description: "Tunda bitta odamni otishi mumkin. Noto'g'ri tanlov shaharga zarar qiladi.", Timing: "night", NightAction: true},
	{Name: RoleJudge, Side: SideTown, Emoji: "Judge", Description: "Ovoz berish natijasiga ta'sir qilishi yoki ayrim qoidalarda ovozni bekor qilishi mumkin.", Timing: "day"},
	{Name: RolePriest, Side: SideTown, Emoji: "Priest", Description: "Yovuz kuchlarni yoki ayrim mustaqil dushmanlarni aniqlashi mumkin.", Timing: "night", NightAction: true},
	{Name: RoleSpy, Side: SideTown, Emoji: "Spy", Description: "Mafia harakatlarini kuzatadi va ba'zan kimga hujum qilinganini biladi.", Timing: "night", NightAction: true},
	{Name: RoleNurse, Side: SideTown, Emoji: "Nurse", Description: "Doctorning cheklanganroq yordamchi turi.", Timing: "night", NightAction: true},
	{Name: RoleMedium, Side: SideTown, Emoji: "Medium", Description: "O'lgan o'yinchilar bilan bog'lana oladi.", Timing: "night", NightAction: true},
	{Name: RoleTracker, Side: SideTown, Emoji: "Tracker", Description: "Tunda tanlangan o'yinchi kimga borganini kuzatadi.", Timing: "night", NightAction: true},
	{Name: RoleWatcher, Side: SideTown, Emoji: "Watcher", Description: "Tanlangan o'yinchining oldiga kim kelganini ko'radi.", Timing: "night", NightAction: true},
	{Name: RoleMafia, Side: SideMafia, Emoji: "Mafia", Description: "Tunda odam o'ldiradi va mafia jamoasi bilan birga ishlaydi.", Timing: "night", NightAction: true},
	{Name: RoleDon, Side: SideMafia, Emoji: "Don", Description: "Mafia boshlig'i. Mafia qarorlarini boshqaradi yoki ayrim qoidalarda tekshiruvlarni chalg'itadi.", Timing: "night", NightAction: true},
	{Name: RoleGodfather, Side: SideMafia, Emoji: "Godfather", Description: "Sheriff tekshirsa ham oddiy odamdek ko'rinishi mumkin.", Timing: "night", NightAction: true},
	{Name: RoleSilencer, Side: SideMafia, Emoji: "Silencer", Description: "Bir o'yinchini kunduz gapirmas qilishi mumkin.", Timing: "night", NightAction: true},
	{Name: RoleFramer, Side: SideMafia, Emoji: "Framer", Description: "Oddiy odamni tekshiruvlarda mafia sifatida ko'rsatishi mumkin.", Timing: "night", NightAction: true},
	{Name: RoleAssassin, Side: SideMafia, Emoji: "Assassin", Description: "Maxsus o'ldirish qobiliyatiga ega mafia roli.", Timing: "night", NightAction: true},
	{Name: RoleBomber, Side: SideMafia, Emoji: "Bomber", Description: "Odamlarga bomba qo'yadi.", Timing: "night", NightAction: true},
	{Name: RolePoisoner, Side: SideMafia, Emoji: "Poisoner", Description: "O'yinchini zaharlaydi; natija keyinroq chiqishi mumkin.", Timing: "night", NightAction: true},
	{Name: RoleNinja, Side: SideMafia, Emoji: "Ninja", Description: "Tekshiruvlarda yoki kuzatuvda ko'rinmaydigan yashirin mafia roli.", Timing: "night", NightAction: true},
	{Name: RoleWerewolf, Side: SideNeutral, Emoji: "Werewolf", Description: "Kechasi odam o'ldiradi. Ayrim qoidalarda mafiadan alohida dushman guruh.", Timing: "night", NightAction: true},
	{Name: RoleJester, Side: SideNeutral, Emoji: "Jester", Description: "Ovoz bilan chiqarib yuborilsa, o'zi g'alaba qiladi.", Timing: "day"},
	{Name: RoleSerialKiller, Side: SideNeutral, Emoji: "Serial Killer", Description: "Har tunda bitta odamni o'ldiradi va yakka o'ynaydi.", Timing: "night", NightAction: true},
	{Name: RoleVampire, Side: SideNeutral, Emoji: "Vampire", Description: "Odamlarni o'z tomoniga o'tkazishi yoki tunda hujum qilishi mumkin.", Timing: "night", NightAction: true},
	{Name: RoleArsonist, Side: SideNeutral, Emoji: "Arsonist", Description: "Odamlarga belgi qo'yib, keyin bir vaqtda yoqib yuborishi mumkin.", Timing: "night", NightAction: true},
	{Name: RoleJoker, Side: SideNeutral, Emoji: "Joker", Description: "Jesterga o'xshash, alohida g'alaba shartlariga ega rol.", Timing: "day"},
	{Name: RoleCultLeader, Side: SideNeutral, Emoji: "Cult", Description: "Odamlarni cult tomoniga o'tkazadi. Ustunlikka erishsa g'alaba qiladi.", Timing: "night", NightAction: true},
	{Name: RoleLover, Side: SideSpecial, Emoji: "Lover", Description: "Boshqa o'yinchi bilan bog'langan bo'ladi; biri o'lsa ikkinchisiga ham ta'sir qilishi mumkin.", Timing: "passive"},
	{Name: RoleCupid, Side: SideSpecial, Emoji: "Cupid", Description: "O'yin boshida sevishganlarni tanlaydi.", Timing: "setup", NightAction: true},
	{Name: RoleTwin, Side: SideSpecial, Emoji: "Twin", Description: "Egizak roli. Birining o'limi boshqasiga ta'sir qilishi mumkin.", Timing: "passive"},
}

func Info(name RoleName) (RoleInfo, bool) {
	for _, info := range Catalog {
		if info.Name == name {
			return info, true
		}
	}
	return RoleInfo{}, false
}

func IsMafiaRole(name RoleName) bool {
	info, ok := Info(name)
	return ok && info.Side == SideMafia
}
