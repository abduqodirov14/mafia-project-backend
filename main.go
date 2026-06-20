package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	database := db.ConnectPostgres(cfg.DatabaseURL)
	userRepo := repositories.NewUserRepository(database)

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("[main] bot init failed: %v", err)
	}
	bot.Debug = false

	botInfo, _ := bot.GetMe()
	log.Printf("[main] bot started: @%s", botInfo.UserName)

	hub := game.NewHub()
	go hub.Run()

	manager := game.NewManager(bot, hub, cfg.AdminChatID, cfg.WebAppURL, botInfo.UserName)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	mux.Handle("/webapp/", http.StripPrefix("/webapp/", http.FileServer(http.Dir("webapp"))))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Demo: create room with bots and auto-start
	mux.HandleFunc("/api/demo/start", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w)

		userIDStr := r.URL.Query().Get("user")
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "Demo"
		}
		var userID int64
		if userIDStr != "" {
			userID, _ = strconv.ParseInt(userIDStr, 10, 64)
		}
		if userID == 0 {
			userID = 99999
		}

		room := manager.CreateRoomWeb(userID, name)

		bots := []struct {
			id   int64
			name string
		}{
			{-1001, "🤖 Sardor"}, {-1002, "🤖 Dilnoza"},
			{-1003, "🤖 Komil"}, {-1004, "🤖 Shaxboz"},
		}
		for _, b := range bots {
			manager.JoinRoom(room.ID, &game.Player{TelegramID: b.id, Username: b.name, IsAlive: true})
		}

		if err := manager.StartGame(room.ID); err != nil {
			sendError(w, err.Error())
			return
		}

		sendJSON(w, map[string]interface{}{
			"room_id":      room.ID,
			"bot_username": botInfo.UserName,
			"webapp_url":   fmt.Sprintf("%s?room=%s", cfg.WebAppURL, room.ID),
			"invite_link":  fmt.Sprintf("https://t.me/%s?start=ref_%s", botInfo.UserName, room.ID),
		})
	})

	// API: Create room
	mux.HandleFunc("/api/room/create", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w)
		if r.Method == http.MethodOptions {
			return
		}

		userIDStr := r.URL.Query().Get("user")
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "O'yinchi"
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			sendError(w, "user ID noto'g'ri")
			return
		}

		room := manager.CreateRoomWeb(userID, name)
		sendJSON(w, map[string]interface{}{
			"room_id":      room.ID,
			"bot_username": botInfo.UserName,
			"invite_link":  fmt.Sprintf("https://t.me/%s?start=ref_%s", botInfo.UserName, room.ID),
		})
	})

	// API: Join room
	mux.HandleFunc("/api/room/join", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w)
		if r.Method == http.MethodOptions {
			return
		}

		var body struct {
			RoomID string `json:"room_id"`
			UserID int64  `json:"user_id"`
			Name   string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			sendError(w, "JSON xato")
			return
		}

		player := &game.Player{TelegramID: body.UserID, Username: body.Name, IsAlive: true}
		if err := manager.JoinRoom(body.RoomID, player); err != nil {
			sendError(w, err.Error())
			return
		}

		room := manager.GetRoom(body.RoomID)
		sendJSON(w, map[string]interface{}{
			"room_id": body.RoomID,
			"count":   room.PlayerCount(),
		})
	})

	// API: Room info
	mux.HandleFunc("/api/room/info", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w)

		roomID := r.URL.Query().Get("room")
		room := manager.GetRoom(roomID)
		if room == nil {
			sendError(w, "xona topilmadi")
			return
		}

		players := make([]map[string]interface{}, 0)
		for _, p := range room.GetPlayerList() {
			players = append(players, map[string]interface{}{
				"id":         p.TelegramID,
				"name":       p.Username,
				"is_alive":   p.IsAlive,
				"join_order": p.JoinOrder,
			})
		}

		sendJSON(w, map[string]interface{}{
			"room_id":   room.ID,
			"owner_id":  room.OwnerID,
			"players":   players,
			"count":     room.PlayerCount(),
			"max":       room.MaxPlayers,
			"status":    string(room.Status),
		})
	})

	// HTTP server
	go func() {
		addr := cfg.ServerAddr()
		log.Printf("[main] http server listening on %s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("[main] server failed: %v", err)
		}
	}()

	// Handlers
	startHandler := handlers.NewStartHandler(bot, cfg, userRepo, manager, botInfo.UserName)
	groupHandler := handlers.NewGroupHandler(bot, cfg, manager, userRepo)
	economyHandler := handlers.NewEconomyHandler(bot, userRepo)
	adminHandler := handlers.NewAdminHandler(bot, cfg, userRepo)
	payHandler := handlers.NewPaymentHandler(bot, userRepo)

	// Bot updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	log.Println("[main] mafia bot tayyor!")

	for update := range updates {
		go processUpdate(update, bot, startHandler, groupHandler, economyHandler, adminHandler, payHandler)
	}
}

func processUpdate(
	update tgbotapi.Update,
	bot *tgbotapi.BotAPI,
	startHandler *handlers.StartHandler,
	groupHandler *handlers.GroupHandler,
	economyHandler *handlers.EconomyHandler,
	adminHandler *handlers.AdminHandler,
	payHandler *handlers.PaymentHandler,
) {
	// Pre-checkout & successful payment
	if update.PreCheckoutQuery != nil || (update.Message != nil && update.Message.SuccessfulPayment != nil) {
		payHandler.Handle(update)
		return
	}

	// Callback queries
	if update.CallbackQuery != nil {
		data := update.CallbackQuery.Data
		if adminHandler.HandleCallback(update.CallbackQuery) {
			return
		}
		if startHandler.HandleCallback(update.CallbackQuery) {
			return
		}
		if payHandler.HandleCallback(update.CallbackQuery) {
			return
		}
		if groupHandler.HandleCallback(update.CallbackQuery) {
			return
		}
		if data == "shop_main" {
			payHandler.Handle(tgbotapi.Update{
				Message: &tgbotapi.Message{
					Chat:     update.CallbackQuery.Message.Chat,
					From:     update.CallbackQuery.From,
					Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Length: 8}},
					Text:     "/buy",
				},
			})
		}
		return
	}

	if update.Message == nil {
		return
	}

	msg := update.Message
	isGroup := msg.Chat.IsGroup() || msg.Chat.IsSuperGroup()

	if isGroup {
		if msg.IsCommand() {
			groupHandler.Handle(update)
		}
		return
	}

	// Private messages
	if msg.IsCommand() {
		cmd := msg.Command()
		switch {
		case adminHandler.IsAdmin(msg.From.ID) && isAdminCommand(cmd):
			adminHandler.Handle(update)
		case isEconomyCommand(cmd):
			economyHandler.Handle(update)
		case isPaymentCommand(cmd):
			payHandler.Handle(update)
		default:
			startHandler.Handle(update)
		}
	}

	if msg.Text == "/+" || msg.Text == "/join" {
		groupHandler.Handle(update)
	}
}

func isAdminCommand(cmd string) bool {
	switch cmd {
	case "admin", "stats", "addcoins", "ban", "broadcast":
		return true
	}
	return false
}

func isEconomyCommand(cmd string) bool {
	switch cmd {
	case "money", "give", "balance", "bal":
		return true
	}
	return false
}

func isPaymentCommand(cmd string) bool {
	switch cmd {
	case "buy", "shop_stars", "donate", "support":
		return true
	}
	return false
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, ngrok-skip-browser-warning")
	w.Header().Set("Content-Type", "application/json")
}

func sendJSON(w http.ResponseWriter, data map[string]interface{}) {
	data["ok"] = true
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": msg})
}
