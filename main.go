package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CustomButton struct {
	Text string
	Link string
}

type AdData struct {
	FileID     string
	Caption    string
	IsVideo    bool
	HasMedia   bool
	Buttons    []CustomButton
	TempButton string
}

type ChannelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserData struct {
	Channels []ChannelInfo `json:"channels"`
}

type GlobalStorage struct {
	Users map[int64]*UserData `json:"users"`
}

var (
	botToken   = "8467228808:AAGQu8TdKykQy2dZlzyY9DD2TklIDwoDe2U"
	adminState = make(map[int64]string)
	userAdData = make(map[int64]*AdData)
)

func main() {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("Bot muvaffaqiyatli ishga tushdi!")

	for update := range updates {
		if update.CallbackQuery != nil {
			handleCallback(bot, update)
			continue
		}
		if update.Message == nil {
			continue
		}
		handleMessage(bot, update)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := update.Message
	userID := msg.From.ID
	chatID := msg.Chat.ID
	text := msg.Text

	if text == "❌ Bekor qilish" || text == "/start" {
		resetUserState(bot, chatID, userID)
		return
	}

	state := adminState[userID]

	switch state {
	case "wait_media":
		handleMediaInput(bot, update)
		return

	case "wait_text":
		if userAdData[userID] == nil {
			userAdData[userID] = &AdData{}
		}
		userAdData[userID].Caption = text
		adminState[userID] = "wait_btn_text"

		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Tomosha qilish"),
				tgbotapi.NewKeyboardButton("Yuklab olish"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("TOMOSHA QILISH"),
				tgbotapi.NewKeyboardButton("YUKLAB OLISH"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("🔹Tomosha qilish🔹"),
				tgbotapi.NewKeyboardButton("🔹Yuklab olish🔹"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("📥 Tomosha qilish"),
				tgbotapi.NewKeyboardButton("📥Yuklab olish"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Yuklab olish📥"),
				tgbotapi.NewKeyboardButton("Tomosha qilish📥"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("✨Tomosha qilish✨"),
				tgbotapi.NewKeyboardButton("✨Yuklab olish✨"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("◁ Tomosha qilish ▷"),
				tgbotapi.NewKeyboardButton("◁ Yuklab olish ▷"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Anime koʻrish"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("❌ Bekor qilish"),
			),
		)
		keyboard.ResizeKeyboard = true

		m := tgbotapi.NewMessage(chatID, "⚙️ **Tugma matnini kiriting:**\n\nPastdagi tayyor variantlardan birini tanlashingiz yoki o'zingiz yozishingiz mumkin.")
		m.ParseMode = "Markdown"
		m.ReplyMarkup = keyboard
		bot.Send(m)
		return

	case "wait_btn_text":
		if userAdData[userID] == nil {
			userAdData[userID] = &AdData{}
		}
		userAdData[userID].TempButton = text
		adminState[userID] = "wait_ad_link"

		m := tgbotapi.NewMessage(chatID, "🔗 Endi tugma linkini yuboring:")
		m.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		bot.Send(m)
		return

	case "wait_ad_link":
		data := userAdData[userID]
		if !strings.HasPrefix(text, "http") {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Xato link! http... bilan boshlansin:"))
			return
		}
		data.Buttons = append(data.Buttons, CustomButton{
			Text: data.TempButton,
			Link: text,
		})
		adminState[userID] = ""
		sendPreview(bot, chatID, userID)
		return

	case "wait_channel_input":
		var cID, cName string
		if msg.ForwardFromChat != nil {
			cID = fmt.Sprintf("%d", msg.ForwardFromChat.ID)
			cName = msg.ForwardFromChat.Title
		} else {
			cID = msg.Text
			cName = msg.Text
		}
		saveToDB(userID, cID, cName)
		adminState[userID] = ""
		bot.Send(tgbotapi.NewMessage(chatID, "✅ "+cName+" muvaffaqiyatli saqlandi!"))
		return

	case "wait_target_channel":
		var targetChatID int64
		var targetChatUsername string

		if msg.ForwardFromChat != nil {
			targetChatID = msg.ForwardFromChat.ID
		} else {
			input := msg.Text
			if strings.HasPrefix(input, "@") {
				targetChatUsername = input
			} else if id, err := strconv.ParseInt(input, 10, 64); err == nil {
				targetChatID = id
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Iltimos, kanalni to'g'ri ko'rsating!"))
				return
			}
		}

		botMember, err := bot.GetChatMember(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID:             targetChatID,
				SuperGroupUsername: targetChatUsername,
				UserID:             bot.Self.ID,
			},
		})
		if err != nil || (!botMember.IsAdministrator() && !botMember.IsCreator()) {
			bot.Send(tgbotapi.NewMessage(chatID, "🚫 **Bot ushbu kanalda admin emas!**"))
			return
		}

		data := userAdData[userID]
		if data == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Reklama ma'lumotlari topilmadi."))
			return
		}

		var rows [][]tgbotapi.InlineKeyboardButton
		for _, b := range data.Buttons {
			btn := tgbotapi.NewInlineKeyboardButtonURL(b.Text, b.Link)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
		}
		keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

		finalChatID := targetChatID
		if finalChatID == 0 {
			chat, _ := bot.GetChat(tgbotapi.ChatInfoConfig{
				ChatConfig: tgbotapi.ChatConfig{SuperGroupUsername: targetChatUsername},
			})
			finalChatID = chat.ID
		}

		var sendTo tgbotapi.Chattable
		if !data.HasMedia {
			m := tgbotapi.NewMessage(finalChatID, data.Caption)
			m.ReplyMarkup = keyboard
			sendTo = m
		} else if data.IsVideo {
			v := tgbotapi.NewVideo(finalChatID, tgbotapi.FileID(data.FileID))
			v.Caption = data.Caption
			v.ReplyMarkup = keyboard
			sendTo = v
		} else {
			p := tgbotapi.NewPhoto(finalChatID, tgbotapi.FileID(data.FileID))
			p.Caption = data.Caption
			p.ReplyMarkup = keyboard
			sendTo = p
		}

		_, err = bot.Send(sendTo)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Xatolik: "+err.Error()))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, "🚀 Reklama muvaffaqiyatli yuborildi!"))
		}
		resetUserState(bot, chatID, userID)
		return
	}

	// Asosiy menyudagi tugmalar
	switch text {
	case "📣 Reklama tayyorlash":
		startAdCreation(bot, chatID, userID)
	}
}

func handleCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	cb := update.CallbackQuery
	data := cb.Data
	userID := cb.From.ID
	chatID := cb.Message.Chat.ID
	messageID := cb.Message.MessageID

	if strings.HasPrefix(data, "select_channel:") {
		idStr := strings.TrimPrefix(data, "select_channel:")
		channelID, _ := strconv.ParseInt(idStr, 10, 64)
		sendAdToChannel(bot, channelID, userID, chatID)
		bot.Request(tgbotapi.NewCallback(cb.ID, "Yuborildi!"))
		return
	}

	if strings.HasPrefix(data, "delete_confirm:") {
		channelIDToDelete := strings.TrimPrefix(data, "delete_confirm:")
		db := loadDB()
		if user, ok := db.Users[userID]; ok {
			var updated []ChannelInfo
			found := false
			for _, ch := range user.Channels {
				if ch.ID != channelIDToDelete {
					updated = append(updated, ch)
				} else {
					found = true
				}
			}
			if found {
				db.Users[userID].Channels = updated
				saveDB(db)
				bot.Request(tgbotapi.NewCallback(cb.ID, "Kanal o'chirildi! ✅"))

				var rows [][]tgbotapi.InlineKeyboardButton
				if len(updated) > 0 {
					for _, ch := range updated {
						btn := tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("❌ "+ch.Name, "delete_confirm:"+ch.ID),
						)
						rows = append(rows, btn)
					}
					rows = append(rows, tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("⬅️ Orqaga", "start_sending"),
					))
					edit := tgbotapi.NewEditMessageText(chatID, messageID, "🗑 Yana birortasini o'chirasizmi?")
					edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
					bot.Send(edit)
				} else {
					edit := tgbotapi.NewEditMessageText(chatID, messageID, "✅ Barcha kanallar o'chirildi.")
					backBtn := tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⬅️ Orqaga", "start_sending")),
					)
					edit.ReplyMarkup = &backBtn
					bot.Send(edit)
				}
				return
			}
		}
	}

	switch data {
	case "add_url_button":
		adminState[userID] = "wait_btn_text"
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Tomosha qilish"),
				tgbotapi.NewKeyboardButton("Yuklab olish"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("TOMOSHA QILISH"),
				tgbotapi.NewKeyboardButton("YUKLAB OLISH"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("🔹Tomosha qilish🔹"),
				tgbotapi.NewKeyboardButton("🔹Yuklab olish🔹"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("📥 Tomosha qilish"),
				tgbotapi.NewKeyboardButton("📥Yuklab olish"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Yuklab olish📥"),
				tgbotapi.NewKeyboardButton("Tomosha qilish📥"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("✨Tomosha qilish✨"),
				tgbotapi.NewKeyboardButton("✨Yuklab olish✨"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("◁ Tomosha qilish ▷"),
				tgbotapi.NewKeyboardButton("◁ Yuklab olish ▷"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Anime koʻrish"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("❌ Bekor qilish"),
			),
		)
		keyboard.ResizeKeyboard = true
		m := tgbotapi.NewMessage(chatID, "⚙️ **Tugma matnini kiriting:**")
		m.ParseMode = "Markdown"
		m.ReplyMarkup = keyboard
		bot.Send(m)

	case "start_sending":
		db := loadDB()
		var rows [][]tgbotapi.InlineKeyboardButton

		if user, ok := db.Users[userID]; ok && len(user.Channels) > 0 {
			for _, ch := range user.Channels {
				btnRow := tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(ch.Name, "select_channel:"+ch.ID),
				)
				rows = append(rows, btnRow)
			}
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Qo'shish", "add_new_to_list"),
		))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 O'chirish", "show_delete_list"),
		))

		m := tgbotapi.NewMessage(chatID, "🔗 Kanalni tanlang:")
		m.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
		bot.Send(m)

	case "add_new_to_list":
		adminState[userID] = "wait_channel_input"
		bot.Send(tgbotapi.NewMessage(chatID, "📥 Kanal ID'sini yuboring yoki kanaldan xabar **Forward** qiling:"))

	case "show_delete_list":
		db := loadDB()
		var rows [][]tgbotapi.InlineKeyboardButton

		if user, ok := db.Users[userID]; ok && len(user.Channels) > 0 {
			for _, ch := range user.Channels {
				btn := tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("❌ "+ch.Name, "delete_confirm:"+ch.ID),
				)
				rows = append(rows, btn)
			}
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Orqaga", "start_sending"),
			))
			edit := tgbotapi.NewEditMessageText(chatID, messageID, "🗑 **Qaysi kanalni o'chirmoqchisiz?**")
			edit.ParseMode = "Markdown"
			edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
			bot.Send(edit)
		} else {
			bot.Request(tgbotapi.NewCallback(cb.ID, "O'chirish uchun kanallar yo'q!"))
		}
	}

	bot.Request(tgbotapi.NewCallback(cb.ID, ""))
}

func sendAdToChannel(bot *tgbotapi.BotAPI, targetChatID int64, userID int64, chatID int64) {
	data := userAdData[userID]
	if data == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Reklama ma'lumotlari topilmadi."))
		return
	}

	if !checkAdmin(bot, targetChatID, bot.Self.ID) {
		bot.Send(tgbotapi.NewMessage(chatID, "🚫 Bot ushbu kanalda admin emas!"))
		return
	}

	if !checkAdmin(bot, targetChatID, userID) {
		bot.Send(tgbotapi.NewMessage(chatID, "🚫 Siz ushbu kanalda admin emassiz!"))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, b := range data.Buttons {
		btn := tgbotapi.NewInlineKeyboardButtonURL(b.Text, b.Link)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	var sendTo tgbotapi.Chattable
	if !data.HasMedia {
		m := tgbotapi.NewMessage(targetChatID, data.Caption)
		m.ReplyMarkup = keyboard
		sendTo = m
	} else if data.IsVideo {
		v := tgbotapi.NewVideo(targetChatID, tgbotapi.FileID(data.FileID))
		v.Caption = data.Caption
		v.ReplyMarkup = keyboard
		sendTo = v
	} else {
		p := tgbotapi.NewPhoto(targetChatID, tgbotapi.FileID(data.FileID))
		p.Caption = data.Caption
		p.ReplyMarkup = keyboard
		sendTo = p
	}

	_, err := bot.Send(sendTo)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Xatolik: "+err.Error()))
	} else {
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🚀 Reklama muvaffaqiyatli yuborildi!\n📍 Kanal: %d", targetChatID)))
		resetUserState(bot, chatID, userID)
	}
}

func checkAdmin(bot *tgbotapi.BotAPI, channelID int64, userID int64) bool {
	member, err := bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: channelID,
			UserID: userID,
		},
	})
	if err != nil {
		return false
	}
	return member.IsAdministrator() || member.IsCreator()
}

func loadDB() GlobalStorage {
	var db GlobalStorage
	db.Users = make(map[int64]*UserData)

	file, err := os.ReadFile("user_channels.json")
	if err != nil {
		return db
	}

	var temp map[string]map[string]UserData
	err = json.Unmarshal(file, &temp)
	if err != nil {
		return db
	}

	if users, ok := temp["users"]; ok {
		for sID, data := range users {
			id, _ := strconv.ParseInt(sID, 10, 64)
			db.Users[id] = &UserData{Channels: data.Channels}
		}
	}
	return db
}

func saveToDB(userID int64, channelID string, channelName string) {
	db := loadDB()
	if _, ok := db.Users[userID]; !ok {
		db.Users[userID] = &UserData{}
	}
	for _, c := range db.Users[userID].Channels {
		if c.ID == channelID {
			return
		}
	}
	db.Users[userID].Channels = append(db.Users[userID].Channels, ChannelInfo{
		ID:   channelID,
		Name: channelName,
	})
	saveDB(db)
}

func saveDB(db GlobalStorage) {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		log.Println("JSON marshaling hatosi:", err)
		return
	}
	os.WriteFile("user_channels.json", data, 0644)
}

func getMainMenu() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📣 Reklama tayyorlash")),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

func getCancelMenu() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("❌ Bekor qilish")),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

func getMediaMenu() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("⏭ Tashlab ketish")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("❌ Bekor qilish")),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

func startAdCreation(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	adminState[userID] = "wait_media"
	msg := tgbotapi.NewMessage(chatID, "📸 Rasm yoki 📹 video yuboring:")
	msg.ReplyMarkup = getMediaMenu()
	bot.Send(msg)
}

func handleMediaInput(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := update.Message
	userID := msg.From.ID
	chatID := msg.Chat.ID

	userAdData[userID] = &AdData{}
	if msg.Photo != nil {
		userAdData[userID].FileID = msg.Photo[len(msg.Photo)-1].FileID
		userAdData[userID].HasMedia = true
		userAdData[userID].IsVideo = false
	} else if msg.Video != nil {
		userAdData[userID].FileID = msg.Video.FileID
		userAdData[userID].HasMedia = true
		userAdData[userID].IsVideo = true
	} else if msg.Text == "⏭ Tashlab ketish" {
		userAdData[userID].HasMedia = false
	} else {
		return
	}

	adminState[userID] = "wait_text"
	resp := tgbotapi.NewMessage(chatID, "✍️ Matnni kiriting:")
	resp.ReplyMarkup = getCancelMenu()
	bot.Send(resp)
}

func sendPreview(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	data := userAdData[userID]

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📤 Uzatish", "start_sending"),
			tgbotapi.NewInlineKeyboardButtonData("➕ Tugma qo'shish", "add_url_button"),
		),
	)

	if !data.HasMedia {
		m := tgbotapi.NewMessage(chatID, data.Caption)
		m.ReplyMarkup = keyboard
		bot.Send(m)
	} else if data.IsVideo {
		v := tgbotapi.NewVideo(chatID, tgbotapi.FileID(data.FileID))
		v.Caption = data.Caption
		v.ReplyMarkup = keyboard
		bot.Send(v)
	} else {
		p := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(data.FileID))
		p.Caption = data.Caption
		p.ReplyMarkup = keyboard
		bot.Send(p)
	}
}

func resetUserState(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	delete(adminState, userID)
	delete(userAdData, userID)
	msg := tgbotapi.NewMessage(chatID, "Salom! 👋\n\nReklama tayyorlash uchun tugmani bosing.")
	msg.ReplyMarkup = getMainMenu()
	bot.Send(msg)
}
