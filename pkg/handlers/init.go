package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jaredallard/balance/pkg/account"
	"github.com/jaredallard/balance/pkg/social"
	log "github.com/sirupsen/logrus"
)

type Handlers struct {
	a *account.Client
}

// NewHandlers creates a new message handler
func NewHandlers(a *account.Client) *Handlers {
	return &Handlers{
		a: a,
	}
}

// HandleNewUser handles an new user event
func (h *Handlers) HandleNewUser(msg *social.Message) (string, error) {
	err := h.a.CreateUser(&account.User{
		PlatformIds: map[account.PlatformName]string{
			msg.PlatformName: msg.UserID,
		},
		PlatformUsernames: map[account.PlatformName]string{
			msg.PlatformName: msg.Username,
		},
	})
	if err != nil {
		return "", err
	}

	return "Hello! I've created you an account. If you need help, or want to know how to use this bot, run /help!", nil
}

func (h *Handlers) HandleAdd(msg *social.Message, tokens []string) (string, error) {
	tokens = tokens[1:]

	balance := 0
	users := make([]account.User, 0)
	for _, user := range tokens {
		var err error
		balance, err = strconv.Atoi(user)
		if err == nil {
			continue
		}

		u, err := h.a.FindUserByUsernam(msg.PlatformName, strings.ToLower(user))
		if err != nil {
			return fmt.Sprintf("Failed to find user %s", user), nil
		}

		users = append(users, *u)
	}

	if balance == 0 {
		return "Balance cannot be 0", nil
	}

	// if we're the only user, don't allow adding ourself
	if len(users) == 1 && users[0].Id == msg.From.Id {
		return "Cannot create a balance with yourself", nil
	}

	log.Infof("creating a balance of '%d' across '%d' users: %v", balance, len(users), users)
	due := float64(balance / len(users))
	for _, user := range users {
		err := h.a.NewTransaction(msg.From, &user, due)
		if err == account.ErrAccountNotFound {
			log.Infof("creating account between user %s and %s", msg.From.Id, user.Id)
			err := h.a.CreateAccount(&account.Account{
				Creator: msg.From,
				Subject: &user,
				Balance: due,
			})
			if err != nil {
				return "Failed to create transaction, please try again later", fmt.Errorf("failed to create account for transaction: %v", err)
			}
		} else if err != nil {
			return "Failed to create transaction, please try again later", fmt.Errorf("failed to create transaction: %v", err)
		}
	}

	if err := h.a.CreateTransaction(msg.From, users, due); err != nil {
		log.Errorf("failed to create transaction log: %v", err)
	}

	return "Balance Created", nil
}

func (h *Handlers) HandleBalance(msg *social.Message) (string, error) {
	accts, err := h.a.FindAccounts(msg.From)
	if err != nil {
		return "Failed to retrieve your balances", err
	}

	m := fmt.Sprintf("Your Balances (%d Accounts):", len(accts))
	for _, a := range accts {
		isNeg := a.Balance < 0
		if isNeg {
			a.Balance = math.Abs(a.Balance)
		}

		creator, err := h.a.GetUser(a.CreatorId)
		if err != nil {
			return "", err
		}

		subject, err := h.a.GetUser(a.SubjectId)
		if err != nil {
			return "", err
		}

		owe := false

		var otherUser *account.User
		if a.CreatorId == msg.From.Id && isNeg {
			owe = true
			otherUser = subject
		} else if a.SubjectId == msg.From.Id && !isNeg {
			owe = true
			otherUser = creator
		}

		if owe {
			m = m + fmt.Sprintf("	You owe %s %v", otherUser.PlatformUsernames[msg.PlatformName], a.Balance)
		} else {
			m = m + fmt.Sprintf("	%s owes you %v", otherUser.PlatformUsernames[msg.PlatformName], a.Balance)
		}
	}
	m = m + "\nTo get my details behind a balance, run /history USERNAME"

	return m, nil
}

// HandleHelp handles /help
func (h *Handlers) HandleHelp(msg *social.Message) (string, error) {
	return fmt.Sprintf(`
Hi! I'm a Bot that will help you track balances between people!

If you want to create a transaction between you and another user, just run /add USERNAME BALANCE

If you want to create a transaction between you and multiple people, run /add USERNAME USERNAME... BALANCE

To view all transactions relating to you, run /history

To view transactions between you and a user, run /history USERNAME

If you need any help, message @jaredallard!
	`), nil
}
