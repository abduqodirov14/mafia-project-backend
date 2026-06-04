package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"mafia-bot/bot/handlers"
	"mafia-bot/config"
	"mafia-bot/db"
	"mafia-bot/db/repositories"
	"mafia-bot/game"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()
	if cfg.BotToken == "" {
		log.Fatal("BOT_TOKEN topilmadi!")
	}

	database := db.ConnectPostgres(cfg.DatabaseURL)
	userRepo := repositories.NewUserRepository(database)

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("Bot xato: %v", err)
	}
	bot.Debug = false
	log.Printf("✅ Bot: @%s", bot.Self.UserName)

	hub := game.NewHub()
	go hub.Run()
	manager := game.NewManager(bot, hub)

	// Hub callbacklarni ulash
	hub.OnConnect      = manager.HandleWebConnect
	hub.OnStartGame    = manager.StartGameByOwner
	hub.OnNightAction  = manager.HandleNightAction
	hub.OnDayVote      = manager.HandleDayVote
	hub.OnConfirmVote  = func(r string, u int64, c bool) { manager.HandleConfirmVote(r, u, c) }

	botInfo, _ := bot.GetMe()
	botUsername := botInfo.UserName

	// ─── HTTP Server ───
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	mux.Handle("/webapp/", http.StripPrefix("/webapp/", http.FileServer(http.Dir("webapp"))))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("OK")) })

	// ─── API: Xona yaratish ───
	mux.HandleFunc("/api/room/create", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == "OPTIONS" { return }
		userIDStr := r.URL.Query().Get("user")
		name := r.URL.Query().Get("name")
		if name == "" { name = "O'yinchi" }
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil { jerr(w, "user ID noto'g'ri"); return }
		room := manager.CreateRoomWeb(userID, name)
		jok(w, map[string]interface{}{
			"room_id":      room.ID,
			"bot_username": botUsername,
			"invite_link":  fmt.Sprintf("https://t.me/%s?start=ref_%s", botUsername, room.ID),
		})
	})

	// ─── API: Qo'shilish ───
	mux.HandleFunc("/api/room/join", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == "OPTIONS" { return }
		var body struct {
			RoomID string `json:"room_id"`
			UserID int64  `json:"user_id"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil { jerr(w, "JSON xato"); return }
		player := &game.Player{TelegramID: body.UserID, Username: body.Name, IsAlive: true}
		if err := manager.JoinRoom(body.RoomID, player); err != nil { jerr(w, err.Error()); return }
		room := manager.GetRoom(body.RoomID)
		jok(w, map[string]interface{}{"room_id": body.RoomID, "count": room.PlayerCount()})
	})

	// ─── API: Xona info ───
	mux.HandleFunc("/api/room/info", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		roomID := r.URL.Query().Get("room")
		room := manager.GetRoom(roomID)
		if room == nil { jerr(w, "xona topilmadi"); return }
		players := []map[string]interface{}{}
		for _, p := range room.GetPlayerList() {
			players = append(players, map[string]interface{}{
				"id": p.TelegramID, "name": p.Username,
				"is_alive": p.IsAlive, "join_order": p.JoinOrder,
			})
		}
		jok(w, map[string]interface{}{
			"room_id": room.ID, "owner_id": room.OwnerID,
			"players": players, "count": room.PlayerCount(),
			"status": string(room.Status),
		})
	})

	go func() {
		port := cfg.ServerPort
		if port == "" { port = "8080" }
		log.Printf("🌐 Server: :%s", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			log.Fatal(err)
		}
	}()

	// ─── Handlers ───
	startHandler   := handlers.NewStartHandler(bot, userRepo, manager)
	roomHandler    := handlers.NewRoomHandler(bot, manager, userRepo, cfg.WebAppURL)
	shopHandler    := handlers.NewShopHandler(bot, userRepo, database)
	gameHandler    := handlers.NewGameHandler(bot, manager)
	payHandler     := handlers.NewPaymentHandler(bot, userRepo)

	// ─── Bot updates ───
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	log.Println("🎮 Mafia bot tayyor! /testgame bilan sinab ko'ring")

	for update := range updates {
		go func(upd tgbotapi.Update) {
			// Pre-checkout va payment
			if upd.PreCheckoutQuery != nil { payHandler.Handle(upd); return }
			if upd.Message != nil && upd.Message.SuccessfulPayment != nil { payHandler.Handle(upd); return }

			// Callback queries
			if upd.CallbackQuery != nil {
				if payHandler.HandleCallback(upd.CallbackQuery) { return }
				gameHandler.HandleCallback(upd)
				shopHandler.Handle(upd)
				return
			}

			// Commands
			if upd.Message != nil && upd.Message.IsCommand() {
				switch upd.Message.Command() {
				case "start", "profile", "rating", "testgame":
					startHandler.Handle(upd)
				case "newroom", "join", "startgame", "leave":
					roomHandler.Handle(upd)
				case "shop", "inventory":
					shopHandler.Handle(upd)
				case "buy", "shop_stars", "donate":
					payHandler.Handle(upd)
				}
			}
		}(update)
	}
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, ngrok-skip-browser-warning")
	w.Header().Set("Content-Type", "application/json")
}
func jok(w http.ResponseWriter, data map[string]interface{}) {
	data["ok"] = true
	json.NewEncoder(w).Encode(data)
}
func jerr(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": msg})
}
