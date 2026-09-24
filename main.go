package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const premiumEmojiID = "5213377637715576188"

const baseLinksFile = "base_links.json"
const maxChannels = 5

// user ID -> tanlangan kanal ID'lari
var selectedChannels = make(map[int64]map[string]bool)

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
	botToken = "8467228808:AAGQu8TdKykQy2dZlzyY9DD2TklIDwoDe2U"
	//botToken   = "8615833296:AAGKXPoj-4BWP2AQWC5jagatjU2QmL12Dd8"
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

		askForLink(bot, chatID, userID)
		return

	case "wait_ad_link":
		data := userAdData[userID]
		if data == nil {
			resetUserState(bot, chatID, userID)
			return
		}
		text = strings.TrimSpace(text)
		if text == "" {
			bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Matn yuboring!"))
			return
		}

		var finalLink string

		if strings.HasPrefix(text, "http") {
			// To'liq link keldi: "=" gacha qismini saqlaymiz
			if base, _, ok := splitLink(text); ok {
				setSavedBase(userID, base)
				bot.Send(tgbotapi.NewMessage(chatID, "💾 Link saqlandi:\n"+base))
			}
			finalLink = text
		} else {
			// Faqat kod keldi: saqlangan linkka qo'shamiz
			base := getSavedBase(userID)
			if base == "" {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Xato link! http... bilan boshlansin:"))
				return
			}
			if strings.ContainsAny(text, " \n\t") {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Kodda bo'sh joy bo'lmasin:"))
				return
			}
			finalLink = base + text
		}

		data.Buttons = append(data.Buttons, CustomButton{
			Text: data.TempButton,
			Link: finalLink,
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
			log.Printf("Xatolik: %v", err)
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Xatolik yuz berdi. Qayta urinib ko'ring."))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, "🚀 Reklama muvaffaqiyatli yuborildi!"))
		}
		resetUserState(bot, chatID, userID)
		return
	}

	// Asosiy menyudagi tugmalar
	// Asosiy menyudagi tugmalar
	switch text {
	case "📣 Reklama tayyorlash":
		startAdCreation(bot, chatID, userID)
	default:
		m := tgbotapi.NewMessage(chatID, "Salom! 👋\n\nReklama tayyorlash uchun tugmani bosing.")
		m.ReplyMarkup = getMainMenu()
		bot.Send(m)
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

	if strings.HasPrefix(data, "toggle_channel:") {
		id := strings.TrimPrefix(data, "toggle_channel:")
		if selectedChannels[userID] == nil {
			selectedChannels[userID] = make(map[string]bool)
		}
		sel := selectedChannels[userID]

		if sel[id] {
			delete(sel, id)
		} else {
			if len(sel) >= maxChannels {
				bot.Request(tgbotapi.NewCallbackWithAlert(cb.ID,
					fmt.Sprintf("Ko'pi bilan %d ta kanal tanlash mumkin!", maxChannels)))
				return
			}
			sel[id] = true
		}

		edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, pickerMarkup(userID))
		bot.Send(edit)
		bot.Request(tgbotapi.NewCallback(cb.ID, ""))
		return
	}

	if strings.HasPrefix(data, "delete_confirm:") {
		channelIDToDelete := strings.TrimPrefix(data, "delete_confirm:")
		db := loadDB()

		user, ok := db.Users[userID]
		if !ok {
			bot.Request(tgbotapi.NewCallbackWithAlert(cb.ID, "Kanallar topilmadi!"))
			return
		}

		var updated []ChannelInfo
		found := false
		for _, ch := range user.Channels {
			if ch.ID == channelIDToDelete {
				found = true
				continue
			}
			updated = append(updated, ch)
		}

		if !found {
			bot.Request(tgbotapi.NewCallbackWithAlert(cb.ID, "Bu kanal ro'yxatda yo'q!"))
			return
		}

		db.Users[userID].Channels = updated
		saveDB(db)

		// Tanlangan kanallar ro'yxatidan ham olib tashlaymiz
		if sel := selectedChannels[userID]; sel != nil {
			delete(sel, channelIDToDelete)
		}

		bot.Request(tgbotapi.NewCallback(cb.ID, "Kanal o'chirildi! ✅"))

		var rows [][]tgbotapi.InlineKeyboardButton
		for _, ch := range updated {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ "+ch.Name, "delete_confirm:"+ch.ID),
			))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ Orqaga", "start_sending"),
		))

		text := "🗑 Yana birortasini o'chirasizmi?"
		if len(updated) == 0 {
			text = "✅ Barcha kanallar o'chirildi."
		}
		edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
		edit.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
		bot.Send(edit)
		return
	}

	if data == "send_selected" {
		sel := selectedChannels[userID]
		if len(sel) == 0 {
			bot.Request(tgbotapi.NewCallbackWithAlert(cb.ID, "Kamida bitta kanal tanlang!"))
			return
		}
		bot.Request(tgbotapi.NewCallback(cb.ID, "Yuborilmoqda..."))

		var report strings.Builder
		var linkRows [][]tgbotapi.InlineKeyboardButton
		report.WriteString("📊 Natija:\n\n")

		db := loadDB()
		if user, ok := db.Users[userID]; ok {
			for _, ch := range user.Channels {
				if !sel[ch.ID] {
					continue
				}

				targetID, err := resolveChannelID(bot, ch.ID)
				if err != nil {
					report.WriteString("❌ " + ch.Name + " — kanal topilmadi\n")
					continue
				}

				link, name, err := postAd(bot, targetID, userID)
				if err != nil {
					report.WriteString("❌ " + ch.Name + " — " + err.Error() + "\n")
				} else {
					report.WriteString("✅ " + name + "\n")
					linkRows = append(linkRows, tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonURL("🔗 "+name, link),
					))
				}
				time.Sleep(400 * time.Millisecond) // Telegram limitiga tushmaslik uchun
			}
		}

		m := tgbotapi.NewMessage(chatID, report.String())
		if len(linkRows) > 0 {
			m.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(linkRows...)
		}
		bot.Send(m)

		resetUserState(bot, chatID, userID)
		return
	}

	switch data {
	case "change_base_link":
		deleteSavedBase(userID)
		adminState[userID] = "wait_ad_link"
		edit := tgbotapi.NewEditMessageText(chatID, messageID,
			"🗑 Eski link o'chirildi.\n\n🔗 Yangi to'liq linkni yuboring.\nMasalan: https://t.me/animlar_uzbekcha_bot?start=1")
		bot.Send(edit)
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
		if selectedChannels[userID] == nil {
			selectedChannels[userID] = make(map[string]bool)
		}
		m := tgbotapi.NewMessage(chatID,
			fmt.Sprintf("🔗 Kanallarni tanlang (%d tagacha):", maxChannels))
		m.ReplyMarkup = pickerMarkup(userID)
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

	sent, err := bot.Send(sendTo)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Xatolik: "+err.Error()))
		return
	}

	postLink := buildPostLink(bot, sent, targetChatID)

	channelName := sent.Chat.Title
	if channelName == "" {
		channelName = strconv.FormatInt(targetChatID, 10)
	}

	m := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"📨 Reklama muvaffaqiyatli yuborildi!\n📌 Kanal: %s\n\n🔗 %s", channelName, postLink))
	m.DisableWebPagePreview = true
	m.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Kanalga o'tish", postLink),
		),
	)
	bot.Send(m)

	resetUserState(bot, chatID, userID)
}

// Kanal tanlash tugmalari
func pickerMarkup(userID int64) tgbotapi.InlineKeyboardMarkup {
	db := loadDB()
	sel := selectedChannels[userID]
	var rows [][]tgbotapi.InlineKeyboardButton

	if user, ok := db.Users[userID]; ok {
		for _, ch := range user.Channels {
			mark := "⬜ "
			if sel[ch.ID] {
				mark = "✅ "
			}
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(mark+ch.Name, "toggle_channel:"+ch.ID),
			))
		}
	}

	if len(sel) > 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("📤 Yuborish (%d)", len(sel)), "send_selected"),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("➕ Qo'shish", "add_new_to_list"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🗑 O'chirish", "show_delete_list"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// "-100123..." yoki "@username" ni int64 ID ga aylantiradi
func resolveChannelID(bot *tgbotapi.BotAPI, s string) (int64, error) {
	if id, err := strconv.ParseInt(s, 10, 64); err == nil {
		return id, nil
	}
	if !strings.HasPrefix(s, "@") {
		s = "@" + s
	}
	chat, err := bot.GetChat(tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{SuperGroupUsername: s},
	})
	if err != nil {
		return 0, err
	}
	return chat.ID, nil
}

// Bitta kanalga yuboradi. Post linki va kanal nomini qaytaradi
func postAd(bot *tgbotapi.BotAPI, targetChatID int64, userID int64) (string, string, error) {
	data := userAdData[userID]
	if data == nil {
		return "", "", fmt.Errorf("reklama ma'lumotlari topilmadi")
	}
	if !checkAdmin(bot, targetChatID, bot.Self.ID) {
		return "", "", fmt.Errorf("bot bu kanalda admin emas")
	}
	if !checkAdmin(bot, targetChatID, userID) {
		return "", "", fmt.Errorf("siz bu kanalda admin emassiz")
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, b := range data.Buttons {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(b.Text, b.Link),
		))
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

	sent, err := bot.Send(sendTo)
	if err != nil {
		return "", "", err
	}

	name := sent.Chat.Title
	if name == "" {
		name = strconv.FormatInt(targetChatID, 10)
	}
	return buildPostLink(bot, sent, targetChatID), name, nil
}

// Kanaldagi yuborilgan postga link yasaydi
func buildPostLink(bot *tgbotapi.BotAPI, sent tgbotapi.Message, channelID int64) string {
	username := sent.Chat.UserName

	// Agar javobda username bo'lmasa, kanal ma'lumotini alohida so'raymiz
	if username == "" {
		chat, err := bot.GetChat(tgbotapi.ChatInfoConfig{
			ChatConfig: tgbotapi.ChatConfig{ChatID: channelID},
		})
		if err == nil {
			username = chat.UserName
		}
	}

	if username != "" {
		return fmt.Sprintf("https://t.me/%s/%d", username, sent.MessageID)
	}

	// Yopiq kanal: -1001234567890 -> 1234567890
	idStr := strings.TrimPrefix(strconv.FormatInt(channelID, 10), "-100")
	return fmt.Sprintf("https://t.me/c/%s/%d", idStr, sent.MessageID)
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
	delete(selectedChannels, userID)
	msg := tgbotapi.NewMessage(chatID, "Salom! 👋\n\nReklama tayyorlash uchun tugmani bosing.")
	msg.ReplyMarkup = getMainMenu()
	bot.Send(msg)
}

// user ID -> "https://t.me/anibla1bot?start="
func loadBaseLinks() map[string]string {
	links := make(map[string]string)
	file, err := os.ReadFile(baseLinksFile)
	if err != nil {
		return links
	}
	json.Unmarshal(file, &links)
	return links
}

func writeBaseLinks(links map[string]string) {
	data, err := json.MarshalIndent(links, "", "  ")
	if err != nil {
		log.Println("JSON marshaling xatosi:", err)
		return
	}
	os.WriteFile(baseLinksFile, data, 0644)
}

func getSavedBase(userID int64) string {
	return loadBaseLinks()[strconv.FormatInt(userID, 10)]
}

func setSavedBase(userID int64, base string) {
	links := loadBaseLinks()
	links[strconv.FormatInt(userID, 10)] = base
	writeBaseLinks(links)
}

func deleteSavedBase(userID int64) {
	links := loadBaseLinks()
	delete(links, strconv.FormatInt(userID, 10))
	writeBaseLinks(links)
}

// "https://t.me/bot?start=600" -> "https://t.me/bot?start=", "600"
func splitLink(link string) (base string, code string, ok bool) {
	i := strings.Index(link, "=")
	if i < 0 {
		return "", "", false
	}
	return link[:i+1], link[i+1:], true
}

// Link bosqichida foydalanuvchidan nima so'rashni hal qiladi
func askForLink(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	base := getSavedBase(userID)

	if base == "" {
		m := tgbotapi.NewMessage(chatID,
			"🔗 Endi to'liq tugma linkini yuboring.\n\n"+
				"Masalan: https://t.me/animlar_uzbekcha_bot?start=1\n\n"+
				"Linkning \"=\" gacha qismi saqlanadi, keyingi safar faqat kodni yuborasiz.")
		m.ReplyMarkup = getCancelMenu()
		bot.Send(m)
		return
	}

	m := tgbotapi.NewMessage(chatID, "🔗 Kodni yuboring (raqam yoki harf):")
	m.ReplyMarkup = getCancelMenu()
	bot.Send(m)

	m2 := tgbotapi.NewMessage(chatID, "💾 Saqlangan link:\n"+base+"KOD")
	m2.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Linkni o'zgartirish", "change_base_link"),
		),
	)
	bot.Send(m2)
}
