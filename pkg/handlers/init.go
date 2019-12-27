package handlers

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/jaredallard/balance/pkg/account"
	"github.com/jaredallard/balance/pkg/social"
	"github.com/pkg/errors"
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
		return "", errors.Wrap(err, "failed to ensure msg.From")
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

	m := fmt.Sprintf("Your Accounts (%d Accounts):\n\n", len(accts))
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
		} else if a.CreatorId == msg.From.Id && !isNeg {
			otherUser = subject
		}

		if a.SubjectId == msg.From.Id && !isNeg {
			owe = true
			otherUser = creator
		} else if a.SubjectId == msg.From.Id && isNeg {
			owe = false
			otherUser = creator
		}

		// TODO(jaredallard): hard dep on USD
		m += " •"
		if owe {
			m = m + fmt.Sprintf("	You owe *%s* $%v", otherUser.PlatformUsernames[msg.PlatformName], a.Balance)
		} else {
			m = m + fmt.Sprintf("	*%s* owes you $%v", otherUser.PlatformUsernames[msg.PlatformName], a.Balance)
		}
		m += "\n"
	}
	m = m + "\nTo get my details behind a balance, run /history USERNAME"

	return m, nil
}

func (h *Handlers) HandleListUsers(msg *social.Message) (string, error) {
	users, err := h.a.ListUsers()
	if err != nil {
		return "", errors.Wrap(err, "failed to list users")
	}

	resp := "Available Users:\n"
	for _, u := range users {
		resp += fmt.Sprintf("• %s\n", u.PlatformUsernames[msg.PlatformName])
	}

	return resp, nil
}

func (h *Handlers) HandleHistory(msg *social.Message, tokens []string) (string, error) {
	userName := ""
	if len(tokens) > 1 {
		userName = tokens[1]
	}
	u, err := h.a.FindUserByUsernam(msg.PlatformName, userName)
	if err != nil && userName != "" { // we ignore empty input since it'll nil the filter
		return "", err
	}

	trans, err := h.a.GetAllTransactionsByUser(msg.From, u)
	if err != nil {
		return "", errors.Wrap(err, "failed to list transactions")
	}

	ctx := ""
	if u != nil {
		ctx = "(" + u.PlatformUsernames[msg.PlatformName] + ")"
	}

	resp := fmt.Sprintf("*Account History %s*\n\n", ctx)
	for _, t := range trans {
		amount := t.Amount / float64(len(t.Accounts))

		// TODO(jaredallard): don't depend on USD
		op := fmt.Sprintf("requested $%v from", amount)

		createdByUser, err := h.a.GetUser(t.CreatedBy)
		if err != nil {
			log.Warnf("failed to show invalid transaction, createdByUser not found: %v", err)
			continue
		}

		if createdByUser.Id == msg.From.Id {
			if u != nil {
				op += " " + u.PlatformUsernames[msg.PlatformName]
			} else {
				for uid := range t.Accounts {
					user, err := h.a.GetUser(uid)
					if err != nil {
						log.Warnf("failed to show invalid transaction, invalid user %s: %v", uid, err)
					}
					op += " " + user.PlatformUsernames[msg.PlatformName]
				}
			}
		} else {
			op += " you"
		}

		str := fmt.Sprintf("_%s_: %s %s", t.CreatedAt.UTC().Format("01-02 15:04"), createdByUser.PlatformUsernames[msg.PlatformName], op)
		resp += str
	}

	return resp, nil
}

// HandleHelp handles /help
func (h *Handlers) HandleHelp(msg *social.Message) (string, error) {
	return fmt.Sprintf(`
Hi! I'm a Bot that will help you track balances between people!

If you want to create a transaction between you and another user, just run /add USERNAME BALANCE

If you want to create a transaction between you and multiple people, run /add USERNAME USERNAME... BALANCE

To view all transactions relating to you, run /history

To view transactions between you and a user, run /history USERNAME

To list all registered users, run /list

To list all account balances, run /status

If you need any help, message @jaredallard!
	`), nil
}
