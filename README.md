# 🎭 MAFIA BOT — To'liq qo'llanma

## O'yin algoritmi ✅

| Rol | Tomon | Vazifa |
|-----|-------|--------|
| 😈 Mafia | Mafia | Har kecha birini o'ldiradi |
| 👑 Don | Mafia | Mafia boshlig'i (7+ o'yinchi) |
| 👨‍⚕️ Doktor | Shahar | Birini davolaydi |
| 🕵️ Sheriff | Shahar | Birini tekshiradi |
| 🛡 Bodyguard | Shahar | Birini himoya qiladi (7+ o'yinchi) |
| 😇 Tinch aholi | Shahar | Mafiyani topadi |

**G'alaba sharti:**
- Shahar yutadi → barcha mafia o'lsa
- Mafia yutadi → mafia soni ≥ shahar soni

---

## Ishga tushirish

### Lokal (test)
```bash
cp .env.example .env
# .env ni to'ldiring
go mod tidy
go run main.go
```

### ngrok bilan test (lokal)
```bash
# 2 ta terminal
go run main.go          # 1-terminal
ngrok http 8080          # 2-terminal

# .env ga qo'shing:
WEBAPP_URL=https://xxxx.ngrok-free.app/webapp
```

---

## 🚀 Deploy (Backend: Railway, Frontend: Vercel)

### 1. GitHub ga yuklash
```bash
git init
git add .
git commit -m "Mafia bot ready"
git remote add origin https://github.com/SIZNING/mafia-bot.git
git push -u origin main
```

### 2. Backend - Railway.app
1. **https://railway.app** → GitHub login
2. **New Project** → **Deploy from GitHub repo** → reponi tanlang
3. **Add Service** → **Database** → **PostgreSQL** qo'shing
4. Bot servisiga **Variables** qo'shing:

```
BOT_TOKEN        = BotFather tokeningiz
DATABASE_URL     = ${{Postgres.DATABASE_URL}}  ← Railway avtomatik
WEBAPP_URL       = https://sizning-webapp.vercel.app/
SERVER_PORT      = 8080
```

5. **Settings** → **Networking** → **Generate Domain**
6. Railway URL'ni ko'chirib oling: `https://sizning-backend.up.railway.app`

### 3. Frontend - Vercel.com
1. **https://vercel.com** → GitHub login
2. **New Project** → **Import** → `webapp` papkasini tanlang
3. **Root Directory**: `webapp` (muhim!)
4. **Deploy** bosing
5. Deploy bo'lgach, Vercel URL'ni oling: `https://sizning-webapp.vercel.app`

### 4. env.js ni yangilash
`webapp/env.js` faylida:
```js
window.API_BASE = 'https://sizning-backend.up.railway.app';
window.APP_URL  = 'https://sizning-webapp.vercel.app';
```

Yangilash:
```bash
cd webapp
git add env.js
git commit -m "Update URLs"
git push
# Vercel avtomatik qayta deploy qiladi
```

### 5. Railway WEBAPP_URL yangilash
Railway'da Variables → WEBAPP_URL:
```
WEBAPP_URL=https://sizning-webapp.vercel.app/
```

### 6. BotFather sozlash
```
/setmenubutton → botingizni tanlang
→ WebApp URL: https://sizning-webapp.vercel.app/
→ Button text: 🎮 O'ynash
```

### 4. Telegram Payments (Stars)
```
/mybots → botingiz → Payments → Stars → Enable
```

---

## Buyruqlar

| Buyruq | Vazifasi |
|--------|----------|
| `/start` | Boshlash |
| `/newroom` | Yangi xona (bot orqali) |
| `/join ID` | Xonaga qo'shilish |
| `/startgame` | O'yinni boshlash |
| `/testgame` | Test (4 bot bilan) |
| `/profile` | Profil |
| `/shop` | Do'kon (tangalar) |
| `/buy` | ⭐ Stars do'koni |
| `/donate` | Yordam berish |
| `/rating` | Reyting |

---

## Daromad olish 💰

Bot **Telegram Stars** orqali pul ishlaydi:

| Mahsulot | Narx |
|----------|------|
| 500 Tanga | 15 ⭐ |
| 1500 Tanga | 40 ⭐ |
| 5000 Tanga | 120 ⭐ |
| 2x XP (7 kun) | 50 ⭐ |
| VIP Nishon | 100 ⭐ |

**1 Telegram Star ≈ $0.013** (Telegram komissiyasi 30%)

Foydalanuvchilar `/buy` buyrug'i orqali xarid qiladi.
Stars hisobingizga tushadigan pul Telegram da ko'rinadi.

---

## Arxitektura

```
mafia-bot/
├── bot/handlers/    — Telegram handlerlari
│   ├── start.go     — /start, /profile, /testgame
│   ├── room.go      — /newroom, /join, /startgame
│   ├── game.go      — Callback handleri
│   ├── shop.go      — /shop, /inventory
│   └── payment.go   — /buy, Telegram Stars
├── config/          — .env sozlamalari
├── db/              — PostgreSQL (GORM)
│   ├── models/      — User, Game, Item modellari
│   └── repositories/ — Ma'lumotlar ombori
├── game/            — O'yin logikasi
│   ├── hub.go       — WebSocket hub (real-vaqt)
│   ├── manager.go   — O'yin boshqaruvi
│   ├── roles/       — Rollar: Mafia, Doktor, Sheriff...
│   └── voice/       — Ovoz boshqaruvi
├── webapp/          — Telegram Mini App
│   └── index.html   — Cinematic UI
└── main.go          — Asosiy fayl + HTTP API
```
