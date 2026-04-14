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

var userState = make(map[int64]string)

type GlobalStats struct {
	TotalUsers     int       `json:"total_users"`      // Botni ishlatgan jami adminlar
	TotalChannels  int       `json:"total_channels"`   // Ulangan jami kanallar
	TotalApproved  int       `json:"total_approved"`   // Shu vaqtgacha jami qabul qilinganlar
	TopChannelName string    `json:"top_channel_name"` // Eng ko'p odam qo'shgan kanal nomi
	MaxApproved    int       `json:"max_approved"`     // O'sha kanal nechta odam qo'shgani
	TotalPosts     int       `json:"total_posts"`      // Shu kanalga yuborilgan jami postlar soni
	LastPostTime   time.Time `json:"last_post_time"`   // Oxirgi post vaqti
}
type ChannelConfig struct {
	OwnerID       int64     `json:"owner_id"`
	ChannelID     int64     `json:"channel_id"`
	ChannelTitle  string    `json:"channel_title"`
	PendingUsers  []int64   `json:"pending_users"` // Kutayotganlar ro'yxati
	TotalApproved int       `json:"total_approved"`
	TotalPosts    int       `json:"total_posts"`
	LastPostTime  time.Time `json:"last_post_time"`
}

type CustomButton struct {
	Text string
	Link string
}

type AdData struct {
	FileID     string
	Caption    string
	IsVideo    bool
	HasMedia   bool
	Buttons    []CustomButton // Tugmalar ro'yxati
	TempButton string         // Vaqtincha matnni saqlab turish uchun (MUHIM)
}

var (
	//botToken = "8534860816:AAEH3QSbf9bj5vr4ARG7tbusvC70WpZgdqY"
	//botToken     = "8615833296:AAHD0Xvoz0HOXv42RFxTVzdPcV1ev7JCJ8E"
	botToken     = "8467228808:AAE6vNO3wu3dvlrnNi2RNy90qwvGp77ErT8"
	adminState   = make(map[int64]string)
	userAdData   = make(map[int64]*AdData)
	channelLinks = make(map[int64]string)
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
		// Kanalga qo'shilish so'rovlarini ushlash
		if update.ChatJoinRequest != nil {
			HandleAutoApprove(bot, update.ChatJoinRequest)
			continue
		}

		if update.CallbackQuery != nil {
			handleCallback(bot, update)
			continue
		}

		if update.Message == nil {
			continue
		}

		handleMessage(bot, update)
	}
	// handleMessage funksiyasi ichida:
}

func handleMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := update.Message
	userID := msg.From.ID
	chatID := msg.Chat.ID
	text := msg.Text

	if text == "❌ Bekor qilish" {
		resetUserState(bot, chatID, userID)
		return
	}
	if text == "/start" {
		resetUserState(bot, chatID, userID)
		return
	}
	// Avval foydalanuvchining holatini (state) aniqlab olamiz
	state, ok := adminState[userID]

	if ok {
		// 1. Link kutish holatini tekshirish (Prefix orqali)
		if strings.HasPrefix(state, "wait_link_") {
			channelIDStr := strings.TrimPrefix(state, "wait_link_")
			channelID, _ := strconv.ParseInt(channelIDStr, 10, 64)

			// Linkni saqlaymiz
			channelLinks[channelID] = text

			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🚀 Ha, yoqish", fmt.Sprintf("start_accept_%d", channelID)),
					tgbotapi.NewInlineKeyboardButtonData("❌ Yo'q", "cancel_accept"),
				),
			)

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Link qabul qilindi: %s\n\nAvto-qabulni yoqamizmi?", text))
			msg.ReplyMarkup = keyboard
			bot.Send(msg)

			// Holatni o'zgartirib qo'yamiz (takroran link yubormasligi uchun)
			adminState[userID] = "confirm_setup"
			return
		}

		// 2. Boshqa holatlarni switch orqali tekshirish
		switch state {
		case "wait_accept_channel":
			SetupJoinRequest(bot, update)
			return
		case "wait_media":
			handleMediaInput(bot, update)
			return
		case "wait_text":
			if userAdData[userID] == nil {
				userAdData[userID] = &AdData{}
			}
			userAdData[userID].Caption = text
			adminState[userID] = "wait_btn_text"

			// 8 ta tayyor tugma variantlari va bekor qilish tugmasi
			keyboard := tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Tomosha qilish"),
					tgbotapi.NewKeyboardButton("Yuklab olish"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("TOMOSHA QILISH"),
					tgbotapi.NewKeyboardButton("YUKLAB OLISH"),
				), tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("🔹Tomosha qilish🔹"),
					tgbotapi.NewKeyboardButton("🔹Yuklab olish🔹"),
				), tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("📥 Tomosha qilish"),
					tgbotapi.NewKeyboardButton("📥Yuklab olish"),
				), tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Yuklab olish📥"),
					tgbotapi.NewKeyboardButton("Tomosha qilish📥"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("✨Tomosha qilish✨"),
					tgbotapi.NewKeyboardButton("✨Yuklab olish✨"),
				), tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("◁ Tomosha qilish ▷"),
					tgbotapi.NewKeyboardButton("◁ Yuklab olish ▷"),
				), tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("Anime koʻrish"),
				),
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton("❌ Bekor qilish"),
				),
			)
			keyboard.ResizeKeyboard = true // Tugmalarni ixcham qilish

			msg := tgbotapi.NewMessage(chatID, "⚙️ **Tugma matnini kiriting:**\n\nPastdagi tayyor variantlardan birini tanlashingiz yoki o'zingiz xohlagan matnni yozib yuborishingiz mumkin.")
			msg.ParseMode = "Markdown"
			msg.ReplyMarkup = keyboard

			bot.Send(msg)
			return
		case "wait_btn_text":
			if userAdData[userID] == nil {
				userAdData[userID] = &AdData{}
			}
			// ButtonText o'rniga TempButton ga saqlaymiz (chunki hali link kelmadi)
			userAdData[userID].TempButton = text

			adminState[userID] = "wait_ad_link"
			msg := tgbotapi.NewMessage(chatID, "🔗 Endi tugma linkini yuboring:")
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			bot.Send(msg)
			return

		case "wait_ad_link":
			data := userAdData[userID]
			if !strings.HasPrefix(text, "http") {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Xato link! http... bilan boshlansin:"))
				return
			}

			// Yangi tugmani ro'yxatga qo'shamiz
			newBtn := CustomButton{
				Text: data.TempButton, // Saqlab qo'yilgan matn
				Link: text,            // Hozir kelgan link
			}
			data.Buttons = append(data.Buttons, newBtn)

			adminState[userID] = ""
			bot.Send(tgbotapi.NewMessage(chatID, "✅ Tugma qo'shildi!"))
			sendPreview(bot, chatID, userID)
			return

		case "start_sending":
			adminState[userID] = "wait_target_channel"
			// Bu yerda @kanal_nomi dagi pastki chiziqni olib tashladik yoki Markdown'ni to'g'irladik
			text := "🔗 **Reklama yuboriladigan kanalni tanlang:**\n\n" +
				"1. Kanaldan birorta xabarni shu yerga **Forward** qiling.\n" +
				"2. Yoki kanal linkini yuboring (masalan: @kanal)\n" +
				"3. Yoki kanal ID raqamini yuboring."

			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = "Markdown" // Yoki bu qatorni umuman o'chirib tashlang, agar format shart bo'lmasa
			bot.Send(msg)
		case "wait_target_channel":
			var targetChatID int64
			var targetChatUsername string

			// 1. Kanalni aniqlash
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

			// 2. Botning adminligini tekshirish
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

			// 3. Foydalanuvchi ma'lumotlarini olish
			data := userAdData[userID]
			if data == nil {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Reklama ma'lumotlari topilmadi."))
				return
			}

			// --- TUGMALARNI TO'G'RI YARATISH (YANGI QISM) ---
			var rows [][]tgbotapi.InlineKeyboardButton
			for _, b := range data.Buttons {
				// Har bir saqlangan tugmani qatorga qo'shamiz
				btn := tgbotapi.NewInlineKeyboardButtonURL(b.Text, b.Link)
				rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
			}
			keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
			// -----------------------------------------------

			// 4. Xabarni yuborish
			var sendTo tgbotapi.Chattable
			finalChatID := targetChatID
			if finalChatID == 0 {
				chat, _ := bot.GetChat(tgbotapi.ChatInfoConfig{
					ChatConfig: tgbotapi.ChatConfig{SuperGroupUsername: targetChatUsername},
				})
				finalChatID = chat.ID
			}

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
				bot.Send(tgbotapi.NewMessage(chatID, "❌ Xatolik: "+err.Error()))
			} else {
				updatePostStats(userID, finalChatID)
				bot.Send(tgbotapi.NewMessage(chatID, "🚀 Reklama muvaffaqiyatli yuborildi!"))
			}

			resetUserState(bot, chatID, userID)
		case "✅ So'rovlarni tasdiqlash":
			bot.Send(tgbotapi.NewMessage(chatID, "Kanal ID-sini yuboring yoki xabarni forward qiling:"))
			adminState[userID] = "wait_for_approve_id"

			// ID kelganda:
			if adminState[userID] == "wait_for_approve_id" {
				targetID, _ := strconv.ParseInt(msg.Text, 10, 64)
				cfg, err := LoadConfig(userID, targetID)

				if err != nil {
					bot.Send(tgbotapi.NewMessage(chatID, "Kanal topilmadi! Avval kanalni sozlang."))
					return
				}

				count := len(cfg.PendingUsers)
				text := fmt.Sprintf("📊 **Kanal:** %s\n👥 Kutayotgan so'rovlar: **%d** ta\n\nBarchasini qabul qilamizmi?",
					cfg.ChannelTitle, count)

				keyboard := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("✅ Hammasini qabul qil", fmt.Sprintf("bulk_approve_%d", targetID)),
					),
				)

				m := tgbotapi.NewMessage(chatID, text)
				m.ReplyMarkup = keyboard
				bot.Send(m)
			}

		case "add_custom_button":
			userState[userID] = "WAITING_BTN_TEXT"
			msg := tgbotapi.NewMessage(chatID, "Tugma matnini kiriting (masalan: Bizga qo'shiling):")
			bot.Send(msg)

		}

	}
	// handleMessage ichida forwardni tutgan joyingizda:
	// 3. Asosiy buyruqlar
	switch text {
	case "/stats":
		// Faqat siz (admin) ko'rishingiz uchun
		if userID == 7518992824 {
			statsText := getHotStats()
			msg := tgbotapi.NewMessage(chatID, statsText)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		}
	case "a":
		if userID == 7518992824 { // Sizning ID ingiz
			text := getHotStats()

			// Agar xohlasangiz, bu yerga faqat admin ko'radigan
			// tugmalarni ham qo'shishingiz mumkin
			bot.Send(tgbotapi.NewMessage(chatID, text))
		}
	case "📣 Reklama tayyorlash":
		startAdCreation(bot, chatID, userID)
	case "🔄 Avto-qabulni sozlash":
		// 1. Avval adminning kanallari bor-yo'qligini tekshiramiz
		files, _ := os.ReadDir("data")
		var foundChannel *ChannelConfig

		for _, file := range files {
			if strings.HasPrefix(file.Name(), fmt.Sprintf("%d_", userID)) {
				// Agar fayl topilsa, uni o'qiymiz
				content, _ := os.ReadFile("data/" + file.Name())
				var cfg ChannelConfig
				json.Unmarshal(content, &cfg)
				foundChannel = &cfg
				break // Hozircha bitta kanalni ko'rib chiqamiz
			}
		}

		// 2. Agar foydalanuvchi hali kanal ulamagan bo'lsa
		if foundChannel == nil {
			startAutoApproveSetup(bot, chatID, userID)
		} else {
			// 3. Agar kanal ulangan bo'lsa, qabul qilish panelini ko'rsatamiz
			count := len(foundChannel.PendingUsers)
			text := fmt.Sprintf("📊 **Kanal:** %s\n👥 Navbatda turganlar: **%d** ta\n\nQabul qilishni boshlaymizmi?",
				foundChannel.ChannelTitle, count)

			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✅ Hammasini qabul qil", fmt.Sprintf("bulk_approve_%d", foundChannel.ChannelID)),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("➕ Yangi kanal qo'shish", "add_new_channel"),
				),
			)

			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = "Markdown"
			msg.ReplyMarkup = keyboard
			bot.Send(msg)
		}

	}
}

func HandleAutoApprove(bot *tgbotapi.BotAPI, request *tgbotapi.ChatJoinRequest) {
	channelID := request.Chat.ID
	userID := request.From.ID

	files, _ := os.ReadDir("data")
	for _, file := range files {
		if strings.HasSuffix(file.Name(), fmt.Sprintf("_%d.json", channelID)) {
			parts := strings.Split(file.Name(), "_")
			ownerID, _ := strconv.ParseInt(parts[0], 10, 64)

			cfg, _ := LoadConfig(ownerID, channelID)

			// Foydalanuvchi allaqachon ro'yxatda bormi tekshiramiz
			exists := false
			for _, id := range cfg.PendingUsers {
				if id == userID {
					exists = true
					break
				}
			}

			if !exists {
				// Faqat ro'yxatga qo'shamiz
				cfg.PendingUsers = append(cfg.PendingUsers, userID)
				cfg.ChannelTitle = request.Chat.Title
				SaveConfig(cfg)
				// Hech qanday tasdiqlash (approve) yuborilmaydi!
			}
		}
	}
}

func SetupJoinRequest(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := update.Message
	var targetChatID int64
	userID := msg.From.ID

	// 1. Kanalni aniqlash (Forward orqali)
	if msg.ForwardFromChat != nil && msg.ForwardFromChat.IsChannel() {
		targetChatID = msg.ForwardFromChat.ID
	} else {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Iltimos, kanaldan biror xabarni forward qiling!"))
		return
	}

	// 2. Foydalanuvchi shu kanalda admin ekanligini tekshirish
	member, err := bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: targetChatID,
			UserID: userID,
		},
	})

	if err != nil || (member.Status != "creator" && member.Status != "administrator") {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "ushbu kanalda admin emassiz yoki meni qoshing!"))
		return
	}

	// 3. Kanal haqida ma'lumot olish (Sorovlar sonini ko'rsatish funksiyasi cheklangan, shuning uchun "Tasdiqlash" so'raymiz)
	chat, _ := bot.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: targetChatID}})

	// Botning texnik imkoniyatini ko'rsatish uchun
	responseText := fmt.Sprintf("📡 **Kanal aniqlandi:** %s\n🆔 ID: `%d`\n\n"+
		"✅ Siz ushbu kanalda adminsiz.\n"+
		"🚀 **Bot imkoniyati:** Soniyasiga ~50-100 ta so'rovni qabul qila oladi.\n\n"+
		"Botni avto-qabul uchun yoqamizmi?",
		chat.Title, targetChatID)

	// 4. Tasdiqlash tugmalari
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Ha, yoqilsin", fmt.Sprintf("approve_%d", targetChatID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Yo'q", "decline"),
		),
	)

	newMsg := tgbotapi.NewMessage(msg.Chat.ID, responseText)
	newMsg.ParseMode = "Markdown"
	newMsg.ReplyMarkup = keyboard
	bot.Send(newMsg)

	// Holatni tozalaymiz (chunki endi link kutmaymiz, tugma bosilishini kutamiz)
	delete(adminState, userID)
}

func handleCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	cb := update.CallbackQuery
	data := cb.Data
	userID := cb.From.ID
	chatID := cb.Message.Chat.ID
	messageID := cb.Message.MessageID

	switch {
	case data == "add_url_button":
		adminState[userID] = "wait_btn_text" // Holatni o'zgartiramiz

		// Foydalanuvchiga xabar yuboramiz
		msg := tgbotapi.NewMessage(chatID, "⚙️ **Tugma matnini kiriting:**\n(Masalan: Tomosha qilish yoki pastdagi variantlardan birini tanlang)")

		// handleMessage dagi o'sha 8 ta tayyor tugmani shu yerda ham ko'rsatish mumkin
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Tomosha qilish"),
				tgbotapi.NewKeyboardButton("Yuklab olish"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("TOMOSHA QILISH"),
				tgbotapi.NewKeyboardButton("YUKLAB OLISH"),
			), tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("🔹Tomosha qilish🔹"),
				tgbotapi.NewKeyboardButton("🔹Yuklab olish🔹"),
			), tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("📥 Tomosha qilish"),
				tgbotapi.NewKeyboardButton("📥Yuklab olish"),
			), tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Yuklab olish📥"),
				tgbotapi.NewKeyboardButton("Tomosha qilish📥"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("✨Tomosha qilish✨"),
				tgbotapi.NewKeyboardButton("✨Yuklab olish✨"),
			), tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("◁ Tomosha qilish ▷"),
				tgbotapi.NewKeyboardButton("◁ Yuklab olish ▷"),
			), tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("Anime koʻrish"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("❌ Bekor qilish"),
			),
		)
		keyboard.ResizeKeyboard = true
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
		// --------------------------------
	// 1. Reklama yuborish qismi (avvalgi mantiqdan)
	case data == "start_sending":
		adminState[userID] = "wait_target_channel"
		text := "🔗 **Reklama yuboriladigan kanalni tanlang:**\n\n" +
			"1. Kanaldan xabarni **Forward** qiling.\n" +
			"2. Yoki kanal ID/Username yuboring."
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		bot.Send(msg)

	// 2. Avto-qabulni rad etish
	case data == "decline" || data == "cancel_accept":
		delete(adminState, userID)
		edit := tgbotapi.NewEditMessageText(chatID, messageID, "🚫 Amaliyot bekor qilindi.")
		bot.Send(edit)

	// 3. Avto-qabulni TASDIQLASH (Sizning logingizdagi approve_ prefiksi uchun)
	case strings.HasPrefix(data, "bulk_approve_"):
		channelID, _ := strconv.ParseInt(strings.TrimPrefix(data, "bulk_approve_"), 10, 64)
		ownerID := cb.From.ID

		cfg, err := LoadConfig(ownerID, channelID)
		if err != nil || len(cfg.PendingUsers) == 0 {
			bot.Send(tgbotapi.NewMessage(chatID, "Kutayotgan so'rovlar topilmadi."))
			return
		}

		total := len(cfg.PendingUsers)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("🚀 %d ta so'rov tasdiqlanmoqda...", total)))

		for _, uID := range cfg.PendingUsers {
			// Har birini bittalab tasdiqlaymiz
			approveReq := tgbotapi.ApproveChatJoinRequestConfig{
				ChatConfig: tgbotapi.ChatConfig{ChatID: channelID},
				UserID:     uID,
			}
			bot.Request(approveReq)
			cfg.TotalApproved++

			// Telegram block qilmasligi uchun kichik pauza
			time.Sleep(time.Millisecond * 100)
		}

		// MUHIM: Qabul qilib bo'lingach, ro'yxatni bo'shatamiz
		cfg.PendingUsers = []int64{}
		SaveConfig(cfg)

		bot.Send(tgbotapi.NewMessage(chatID, "✅ Barcha so'rovlar qabul qilindi. Navbat tozalandi!"))

	case data == "add_new_channel":
		// Bu yerda o'sha forward qilishni so'raydigan funksiyani chaqiramiz
		startAutoApproveSetup(bot, chatID, userID)

	case strings.HasPrefix(data, "approve_"):
		channelIDStr := strings.TrimPrefix(data, "approve_")
		channelID, _ := strconv.ParseInt(channelIDStr, 10, 64)
		ownerID := cb.From.ID

		// Yangi kanal uchun bo'sh konfig yaratamiz
		cfg := ChannelConfig{
			OwnerID:       ownerID,
			ChannelID:     channelID,
			ChannelTitle:  "Kanal",   // Buni keyinchalik HandleAutoApprove yangilab oladi
			PendingUsers:  []int64{}, // Bo'sh ro'yxat
			TotalApproved: 0,
		}

		// JSON faylga saqlaymiz (data/userID_channelID.json)
		SaveConfig(cfg)

		// Ekranni yangilaymiz
		edit := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID,
			"✅ **Kanal muvaffaqiyatli ulandi!**\n\nEndi ushbu kanalga keladigan barcha qo'shilish so'rovlari navbatga yig'iladi.")
		edit.ParseMode = "Markdown"
		bot.Send(edit)
		// Callback aylanib turmasligi uchun javob beramiz

	}

	// Tugmadagi "yuklanish" aylanasini to'xtatish
	bot.Request(tgbotapi.NewCallback(cb.ID, ""))
}

func getMainMenu() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📣 Reklama tayyorlash")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🔄 Avto-qabulni sozlash")),
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

func startAutoApproveSetup(bot *tgbotapi.BotAPI, chatID int64, userID int64) {
	adminState[userID] = "wait_accept_channel"
	text := "🔄 **Avto-qabulni sozlash uchun:**\n\n1. Botni kanalingizga admin qiling.\n2. Kanaldan xabarni forward qiling."
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = getCancelMenu()
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

	// Ikkita tugmali klaviatura: "Uzatish" va "Tugma qo'shish"
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
	msg := tgbotapi.NewMessage(chatID, "Salom, salom! Adminlar\n\n"+
		"Bot yangilandi!\n"+
		"Hosh sinab koring kamchilik bolsa @Hao_aniuz")
	msg.ReplyMarkup = getMainMenu()
	bot.Send(msg)
}

func SaveConfig(cfg ChannelConfig) {
	// 'data' papkasi borligini tekshirish
	_ = os.Mkdir("data", 0755)

	fileName := fmt.Sprintf("data/%d_%d.json", cfg.OwnerID, cfg.ChannelID)
	file, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(fileName, file, 0644)
}

func LoadConfig(ownerID, channelID int64) (ChannelConfig, error) {
	fileName := fmt.Sprintf("data/%d_%d.json", ownerID, channelID)
	file, err := os.ReadFile(fileName)
	if err != nil {
		return ChannelConfig{}, err
	}
	var cfg ChannelConfig
	_ = json.Unmarshal(file, &cfg)
	return cfg, nil
}

func getHotStats() string {
	files, _ := os.ReadDir("data")

	uniqueUsers := make(map[int64]bool)
	totalChannels := 0
	totalPending := 0
	totalApproved := 0

	var topChannelName string
	maxApproved := -1

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			content, _ := os.ReadFile("data/" + file.Name())
			var cfg ChannelConfig
			json.Unmarshal(content, &cfg)

			uniqueUsers[cfg.OwnerID] = true
			totalChannels++
			totalPending += len(cfg.PendingUsers)
			totalApproved += cfg.TotalApproved

			// Eng aktiv kanalni aniqlash
			if cfg.TotalApproved > maxApproved {
				maxApproved = cfg.TotalApproved
				topChannelName = cfg.ChannelTitle
			}
		}
	}

	// Ma'lumotlarni JSON faylga saqlab qo'yamiz (keyinchalik tezkor ko'rish uchun)
	global := GlobalStats{
		TotalUsers:     len(uniqueUsers),
		TotalChannels:  totalChannels,
		TotalApproved:  totalApproved,
		TopChannelName: topChannelName,
		MaxApproved:    maxApproved,
	}
	jsonData, _ := json.MarshalIndent(global, "", "  ")
	os.WriteFile("stats.json", jsonData, 0644)

	// Siz so'ragan formatda qaytarish
	return fmt.Sprintf("🔥 Botning HOT statistikasi:\n\n"+
		"👥 Aktiv Adminlar: %d ta\n"+
		"📢 Jami kanallar: %d ta\n"+
		"✅ Jami qabul qilinganlar: %d ta\n"+
		"⏳ Hozir navbatda: %d ta\n\n"+
		"🏆 Eng aktiv kanal: %s\n"+
		"📈 Muvaffaqiyatli qabul: %d ta",
		len(uniqueUsers), totalChannels, totalApproved, totalPending, topChannelName, maxApproved)
}

func updatePostStats(ownerID int64, channelID int64) {
	cfg, err := LoadConfig(ownerID, channelID)
	if err == nil {
		cfg.TotalPosts++              // Endi bu xato bermaydi
		cfg.LastPostTime = time.Now() // Endi bu ham ishlaydi
		SaveConfig(cfg)
	}
}
