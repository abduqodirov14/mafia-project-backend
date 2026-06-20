package game

import (
	"fmt"
	"math/rand"
	"strings"

	"mafia-bot/game/roles"
)

var nightStartMessages = []string{
	"🌙 <b>Qorong'i tushdi...</b>\nShahar uyquga ketdi. Ammo kimdir yomonlik rejalar qilmoqda...",
	"🌙 <b>TUN %d BOSHLANDI</b>\nKo'chalar bo'shab qoldi. Faqat qorong'i va sirlar qoldi...",
	"🌑 <b>Shahar uyquda...</b>\nAmmo tunning qorong'iligida qotillar uyg'oq...",
}

var dayStartMessages = []string{
	"☀️ Quyosh chiqib, tunda to'kilgan qonlarni quritdi...",
	"☀️ <b>KUN %d BOSHLANDI</b>\nAholi tunda bo'lgan voqealarni muhokama qilmoqda...",
	"🌅 Yana bir kun boshlandi. Kecha nimalar bo'ldi — hamma bilmoqchi...",
}

func NightStartMsg(round int) string {
	msg := nightStartMessages[rand.Intn(len(nightStartMessages))]
	return fmt.Sprintf(msg, round)
}

func DayStartMsg(round int) string {
	msg := dayStartMessages[rand.Intn(len(dayStartMessages))]
	return fmt.Sprintf(msg, round)
}

func DeathMessage(player *Player, killer roles.RoleName) string {
	roleInfo := roles.Get(player.Role)

	deathVerb := "o'ldirildi"
	switch killer {
	case roles.RoleMafia, roles.RoleDon:
		deathVerb = "vaxshiylarcha o'ldirildi"
	case roles.RoleManyak:
		deathVerb = "manyak tomonidan o'ldirildi"
	case roles.RoleKamikaze:
		deathVerb = "portlash natijasida halok bo'ldi"
	}

	msg := fmt.Sprintf("🔴 Tunda %s <b>%s</b> %s...", roleInfo.Emoji, player.Username, deathVerb)

	lastWords := []string{
		"Men o'yin paytida boshqa uxlamayma-a-a-a-a-a-an!",
		"Bu adolatsizlik! Men begunohman!!!",
		"Siz pushaymon bo'lasizlar...",
		"Hali ko'rasizlar meni kim ekanimni!",
	}
	msg += fmt.Sprintf("\n\n💬 O'limidan oldin kimdir <b>%s</b> qichqirganini eshitdi:\n<i>%s</i>",
		player.Username, lastWords[rand.Intn(len(lastWords))])

	return msg
}

func VoteOutMessage(player *Player) string {
	roleInfo := roles.Get(player.Role)
	msgs := []string{
		"⚖️ Shahar aholisi <b>%s</b> ni xiyonatkor deb topib, shahardan quvib chiqardi.\nUning roli: %s %s",
		"🗳 Ovoz berish yakuni: <b>%s</b> chiqarib yuborildi.\nRoli: %s %s",
		"👋 <b>%s</b> shahardan haydaldi.\nRoli: %s %s",
	}
	return fmt.Sprintf(msgs[rand.Intn(len(msgs))], player.Username, roleInfo.Emoji, string(player.Role))
}

func NoVoteMessage() string {
	msgs := []string{
		"🤝 Aholi kelisha olmadi — hech kim chiqarilmadi.",
		"🤷 Ovozlar teng bo'lib qoldi. Bugun hech kim ketmadi.",
		"😮‍💨 Shahar bugun qaror qabul qila olmadi...",
	}
	return msgs[rand.Intn(len(msgs))]
}

func SavedMessage(playerName string) string {
	msgs := []string{
		"✨ Kechasi kimdir hujumdan omon qoldi! Shifokor o'z ishini qildi.",
		"💊 <b>%s</b> hujumga uchradi, lekin shifokor uni qutqardi!",
	}
	msg := msgs[rand.Intn(len(msgs))]
	if strings.Contains(msg, "%s") {
		return fmt.Sprintf(msg, playerName)
	}
	return msg
}

func KomissarResultMsg(target *Player, isMafia bool) string {
	roleInfo := roles.Get(target.Role)
	if isMafia {
		return fmt.Sprintf("🚨 <b>Komissar natijasi:</b>\n<b>%s</b> — MAFIA! %s %s",
			target.Username, roleInfo.Emoji, string(target.Role))
	}
	return fmt.Sprintf("✅ <b>Komissar natijasi:</b>\n<b>%s</b> — Tinch aholi %s",
		target.Username, roleInfo.Emoji)
}

func DaydiResultMsg(target *Player, visitors []*Player) string {
	if len(visitors) == 0 {
		return fmt.Sprintf("🧙🏻‍♂️ <b>Daydi</b> <b>%s</b> uyida tunadi.\nKecha unikiga hech kim kelmadi.", target.Username)
	}
	names := make([]string, len(visitors))
	for i, v := range visitors {
		info := roles.Get(v.Role)
		names[i] = fmt.Sprintf("%s %s", info.Emoji, v.Username)
	}
	return fmt.Sprintf("🧙🏻‍♂️ <b>Daydi</b> <b>%s</b> uyida tunadi.\nKecha unikiga kelganlar: %s",
		target.Username, strings.Join(names, ", "))
}

func DonInheritMessage(newDon *Player) string {
	return fmt.Sprintf("🤵🏻 <b>Don</b> vafot etdi.\n🤵🏼 <b>%s</b> yangi Don bo'ldi!", newDon.Username)
}

func KomissarInheritMessage(newKomissar *Player) string {
	return fmt.Sprintf("🕵🏼 <b>Komissar</b> vafot etdi.\n👮‍♂️ <b>%s</b> (Serjant) Komissar vazifasini oldi!", newKomissar.Username)
}

func KamikazeDeathMessage(kamikaze *Player, target *Player) string {
	tRoleInfo := roles.Get(target.Role)
	return fmt.Sprintf("💣 <b>Kamikaze %s</b> otib o'ldirildi!\nPortlash natijasida <b>%s</b> (%s %s) ham halok bo'ldi!",
		kamikaze.Username, target.Username, tRoleInfo.Emoji, string(target.Role))
}

func NobodyDiedMsg() string {
	msgs := []string{
		"✨ Kechasi hech kim o'lmadi! Shahar yana bir kun xotirjam.",
		"🍀 Kecha hamma omon qoldi!",
		"🌟 Tunda hech qanday qon to'kilmadi.",
	}
	return msgs[rand.Intn(len(msgs))]
}

func WinMessage(winner string, players []*Player) string {
	var title, subtitle string
	switch winner {
	case "mafia":
		title = "😈 <b>MAFIA G'ALABA QOZONDI!</b>"
		subtitle = "Shahar mafia nazoratiga o'tdi..."
	case "town":
		title = "🎉 <b>TINCH AHOLI G'ALABA QOZONDI!</b>"
		subtitle = "Barcha mafiyachilar yo'q qilindi! Shahar xavfsiz!"
	case "manyak":
		title = "🔪 <b>MANYAK G'ALABA QOZONDI!</b>"
		subtitle = "Hamma o'ldi, faqat u tirik qoldi..."
	case "suidsid":
		title = "🧌 <b>SUIDSID G'ALABA QOZONDI!</b>"
		subtitle = "Aholi uni chiqarib yubordi — va u yutdi!"
	default:
		title = "🏁 <b>O'YIN TUGADI</b>"
	}

	msg := title
	if subtitle != "" {
		msg += "\n" + subtitle
	}
	msg += "\n\n<b>O'yinchilar rollari:</b>\n"
	for _, p := range players {
		info := roles.Get(p.Role)
		status := "☠️"
		if p.IsAlive {
			status = "✅"
		}
		msg += fmt.Sprintf("%s %s %s — %s\n", status, info.Emoji, p.Username, string(p.Role))
	}
	return msg
}

func JoinMessage(count, max int) string {
	progressBar := strings.Repeat("▓", count) + strings.Repeat("░", max-count)
	return fmt.Sprintf("🎭 <b>O'yin boshlandi!</b>\n\n"+
		"Qo'shilish uchun quyidagi tugmani bosing:\n\n"+
		"👥 O'yinchilar: <b>%d/%d</b>\n"+
		"[%s]\n\n"+
		"Kamida <b>4 kishi</b> kerak", count, max, progressBar)
}

func PlayerJoinedMsg(username string, count, max int) string {
	return fmt.Sprintf("✅ <b>%s</b> o'yinga qo'shildi!\n👥 O'yinchilar: <b>%d/%d</b>", username, count, max)
}

func PlayerLeftMsg(username string, count, max int) string {
	return fmt.Sprintf("👋 <b>%s</b> o'yindan chiqdi.\n👥 O'yinchilar: <b>%d/%d</b>", username, count, max)
}

func RolePrivateMsg(player *Player) string {
	info := roles.Get(player.Role)
	return fmt.Sprintf("🎭 <b>Sizning rolingiz:</b>\n\n%s <b>%s</b>\n\n<i>%s</i>",
		info.Emoji, string(player.Role), info.Description)
}

func VotingMsg(players []*Player) string {
	msg := "🗳 <b>OVOZ BERISH VAQTI</b>\n\nKimni chiqarib yuboramiz? (60 soniya)\n\n"
	msg += "<b>Tirik o'yinchilar:</b>\n"
	for i, p := range players {
		msg += fmt.Sprintf("%d. %s\n", i+1, p.Username)
	}
	return msg
}
