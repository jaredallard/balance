package telegram

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jaredallard/balance/pkg/account"
	"github.com/jaredallard/balance/pkg/social"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

type Provider struct {
	client  *tgbotapi.BotAPI
	account *account.Client
	cache   *cache.Cache
}

// NewProvider creates a new Telegram message provider
func NewProvider(a *account.Client) (*Provider, error) {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		return nil, err
	}

	c := cache.New(30*time.Minute, 1*time.Hour)

	return &Provider{
		client:  bot,
		account: a,
		cache:   c,
	}, nil
}

func (p *Provider) processUpdate(update tgbotapi.Update, stream chan social.Message) error {
	log.Infof("got update: %v", update)
	if update.Message == nil { // ignore any non-Message Updates
		log.Infof("skipping non-message update")
		return nil
	}

	username := update.Message.From.UserName
	if username == "" {
		username = update.Message.From.FirstName + update.Message.From.LastName
	}
	username = strings.ToLower(username)

	msg := social.Message{
		ChatID:       strconv.Itoa(int(update.Message.Chat.ID)),
		Username:     username,
		UserID:       strconv.Itoa(update.Message.From.ID),
		PlatformName: account.PlatformTelegram,
		Text:         update.Message.Text,
		Replyer: func(chatId, text string) error {
			chatID, err := strconv.Atoi(chatId)
			if err != nil {
				return err
			}

			log.Infof("[telegram] sending message: %v", strings.ReplaceAll(text, "\n", "\\n"))

			msg := tgbotapi.NewMessage(int64(chatID), text)
			msg.ReplyToMessageID = update.Message.MessageID
			msg.ParseMode = "Markdown"
			_, err = p.client.Send(msg)
			return err
		},
	}

	cacheKey := fmt.Sprintf("%s:%d", account.PlatformTelegram, update.Message.From.ID)
	v, found := p.cache.Get(cacheKey)

	// check if we didn't find a user
	if !found || v == nil {
		log.Warnf("cache miss for user: %s", cacheKey)
		u, err := p.account.FindUser(account.PlatformTelegram, strconv.Itoa(update.Message.From.ID))
		if err != nil {
			msg.Error = err
		} else {
			msg.From = u
		}
		p.cache.Set(cacheKey, u, cache.DefaultExpiration)
	} else { // we found the user in our cache
		msg.From = v.(*account.User)
	}

	// TODO(jaredallard): cache users
	stream <- msg
	return nil
}

// CreatStream returns a telegram message stream
func (p *Provider) CreateStream(ctx context.Context) (<-chan social.Message, error) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := p.client.GetUpdatesChan(u)
	if err != nil {
		return nil, err
	}

	stream := make(chan social.Message)

	go func() {
		// close the channel once we're done
		defer close(stream)

		for {
			select {
			case update := <-updates:
				if err := p.processUpdate(update, stream); err != nil {
					log.Errorf("error processing message: %v", err)
				}
			case <-ctx.Done():
				log.Warnf("message processor shutdown")
				return
			}
		}
	}()

	return stream, nil
}

func (p *Provider) Send(m *social.Message) error {
	chatID, err := strconv.Atoi(m.ChatID)
	if err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(int64(chatID), m.Text)
	msg.ParseMode = "Markdown"
	_, err = p.client.Send(msg)
	return err
}
