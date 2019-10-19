package social

import (
	"github.com/jaredallard/balance/pkg/account"
)

// Message is a message published be a social media provider
type Message struct {
	// from is who this message is from
	From *account.User

	// ChatID is the underlying provider's chatId
	ChatID string

	// Username is the username of the user we got this from
	Username string

	// UserID is the underlying provider's userId
	UserID string

	// PlatformName is the social media platform this came from
	PlatformName account.PlatformName

	Replyer func(chatId, message string) error

	// Error is included if an error occurred while processing this message
	Error error

	// Text is the message text
	Text string
}

// Reply is an easier to use interface for the built-in message replyer
func (m *Message) Reply(text string) error {
	return m.Replyer(m.ChatID, text)
}

// Provider is a Social Media provider that integrates with an account
type Provider interface {
	// CreateStream returns a message stream from a provider
	CreateStream() (<-chan *Message, error)
}
