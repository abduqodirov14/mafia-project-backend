package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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

	// DB
	database := db.ConnectPostgres(cfg.DatabaseURL)
	userRepo := repositories.NewUserRepository(database)

	// Bot
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("Bot xato: %v", err)
	}
	bot.Debug = false
	botInfo, _ := bot.GetMe()
	log.Printf("✅ Bot: @%s", botInfo.UserName)

	// Hub + Manager
	hub := game.NewHub()
	go hub.Run()

	manager := game.NewManager(bot, hub, cfg.AdminChatID, cfg.WebAppURL, botInfo.UserName)

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
			"bot_username": botInfo.UserName,
			"invite_link":  fmt.Sprintf("https://t.me/%s?start=ref_%s", botInfo.UserName, room.ID),
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
			"max": room.MaxPlayers, "status": string(room.Status),
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
	startHandler   := handlers.NewStartHandler(bot, userRepo, manager, cfg.WebAppURL, botInfo.UserName)
	groupHandler   := handlers.NewGroupHandler(bot, manager, userRepo, cfg.WebAppURL)
	economyHandler := handlers.NewEconomyHandler(bot, userRepo)
	adminHandler   := handlers.NewAdminHandler(bot, userRepo)
	payHandler     := handlers.NewPaymentHandler(bot, userRepo)

	// ─── Updates ───
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	log.Println("🎮 Mafia bot tayyor! Guruhga qo'shing va /start yuboring")

	for update := range updates {
		go func(upd tgbotapi.Update) {
			// Pre-checkout
			if upd.PreCheckoutQuery != nil {
				payHandler.Handle(upd)
				return
			}
			// To'lov
			if upd.Message != nil && upd.Message.SuccessfulPayment != nil {
				payHandler.Handle(upd)
				return
			}

			// Callback queries
			if upd.CallbackQuery != nil {
				data := upd.CallbackQuery.Data

				// Admin callbacks
				if adminHandler.HandleCallback(upd.CallbackQuery) { return }
				// Start callbacks
				if startHandler.HandleCallback(upd.CallbackQuery) { return }
				// Pay callbacks
				if payHandler.HandleCallback(upd.CallbackQuery) { return }
				// Group callbacks (join, leave, vote, night actions)
				if groupHandler.HandleCallback(upd.CallbackQuery) { return }

				// Shop callbacks
				if data == "shop_main" {
					payHandler.Handle(tgbotapi.Update{
						Message: &tgbotapi.Message{
							Chat: upd.CallbackQuery.Message.Chat,
							From: upd.CallbackQuery.From,
							Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Length: 8}},
							Text: "/buy",
						},
					})
				}
				return
			}

			if upd.Message == nil {
				return
			}
			msg := upd.Message
			isGroup := msg.Chat.IsGroup() || msg.Chat.IsSuperGroup()

			// Guruh xabarlari
			if isGroup {
				if msg.IsCommand() {
					groupHandler.Handle(upd)
				}
				return
			}

			// Shaxsiy xabarlar
			if msg.IsCommand() {
				cmd := msg.Command()
				switch {
				case adminHandler.IsAdmin(msg.From.ID) &&
					(cmd == "admin" || cmd == "stats" || cmd == "addcoins" || cmd == "ban" || cmd == "broadcast"):
					adminHandler.Handle(upd)

				case cmd == "money" || cmd == "give" || cmd == "balance" || cmd == "bal":
					economyHandler.Handle(upd)

				case cmd == "buy" || cmd == "shop_stars" || cmd == "donate" || cmd == "support":
					payHandler.Handle(upd)

				default:
					startHandler.Handle(upd)
				}
			}

			// /+ va /- shaxsiy chatda
			if msg.Text == "/+" || strings.HasPrefix(msg.Text, "/join") {
				groupHandler.Handle(upd)
			}
		}(update)
	}
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
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
