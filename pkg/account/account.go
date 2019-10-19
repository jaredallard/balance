package account

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/gofrs/uuid"
)

var (
	// ErrAccountNotFound is returned when an account doesn't exist
	ErrAccountNotFound error = errors.New("Account not found")

	// ErrAccountExists is returned when an account is attempted to be created, but already exists
	ErrAccountExists error = errors.New("Account already exists")
)

// Account is a user's account
type Account struct {
	// Id of this account
	Id uuid.UUID `json:"id" pg:",pk,type:uuid,default:uuid_generate_v4()"`

	CreatorId uuid.UUID `json:"creator_id" pg:"type:uuid,notnull"`
	// Creator is who owns this account, and what the balance is reflective of
	Creator *User `json:"creator"`

	SubjectId uuid.UUID `json:"subject_id" pg:"type:uuid,notnull"`
	// Subject is who this account includes, who is related to it, etc.
	Subject *User `json:"subject"`

	// Balance is the current balance of this account
	Balance float64 `json:"balance" pg:"default:0"`

	CreatedAt time.Time `json:"created_at" pg:"default:now(),notnull"`
	UpdatedAt time.Time `json:"updated_at" pg:"default:now(),notnull"`
}

func (a *Account) String() string {
	return fmt.Sprintf("Account<ID: %s, Creator: %s, Subject: %s, Balance: %v>", a.Id, a.Creator, a.Subject, a.Balance)
}

type Client struct {
	db *pg.DB
}

// NewClient creates a new client
func NewClient(db *pg.DB) *Client {
	return &Client{
		db: db,
	}
}

// FindAccounts finds all accounts owned by a user
func (c *Client) FindAccounts(u *User) ([]Account, error) {
	var a []Account
	err := c.db.Model(&a).
		Relation("Creator").
		Relation("Subject").
		Where("account.creator_id = ? OR account.subject_id = ?", u.Id, u.Id).
		Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, ErrAccountNotFound
	}

	return a, err
}

// FindAccountBetween finds an account between a user
func (c *Client) FindAccountBetween(u1 *User, u2 *User) (*Account, error) {
	a := &Account{}
	err := c.db.Model(a).
		Relation("Creator").
		Relation("Subject").
		Where("(account.creator_id = ? AND account.subject_id = ?) OR (account.creator_id = ? AND account.subject_id = ?)", u1.Id, u2.Id, u2.Id, u1.Id).
		Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, ErrAccountNotFound
	}

	return a, err
}

// NewTransaction records a new transaction
func (c *Client) NewTransaction(creator *User, subject *User, amount float64) error {
	a, err := c.FindAccountBetween(creator, subject)
	if err != nil {
		return err
	}

	newBalance := a.Balance
	isNeg := amount < 0

	if isNeg {
		amount = math.Abs(amount)
	}

	if a.CreatorId == creator.Id {
		if isNeg {
			newBalance = newBalance - amount
		} else {
			newBalance = newBalance + amount
		}
	} else if a.SubjectId == creator.Id {
		if isNeg {
			newBalance = newBalance + amount
		} else {
			newBalance = newBalance - amount
		}
	} else {
		return ErrAccountNotFound
	}

	a.Balance = newBalance
	a.UpdatedAt = time.Now()

	_, err = c.db.Model(a).Column("balance", "updated_at").Where("account.id = ?", a.Id).Update(a)
	return err
}

// GetAccount returns an account
func (c *Client) GetAccount(id uuid.UUID) (*Account, error) {
	a := &Account{Id: id}
	err := c.db.Model(a).
		Relation("Creator").
		Relation("Subject").
		Where("account.id = ?", a.Id).
		Select(a)
	if errors.Is(err, pg.ErrNoRows) {
		return nil, ErrAccountNotFound
	}

	return a, err
}

// CreateAccount creates an account
func (c *Client) CreateAccount(a *Account) error {
	// we have to set a.CreatorId to get our ORM to properly create the relations
	if a.Creator != nil && a.CreatorId != a.Creator.Id {
		a.CreatorId = a.Creator.Id
	}

	if a.Subject != nil && a.SubjectId != a.Subject.Id {
		a.SubjectId = a.Subject.Id
	}

	if a.CreatorId.String() == "" || a.SubjectId.String() == "" {
		return fmt.Errorf("An account must have a creatorId and subjectId")
	}

	_, err := c.db.Model(a).Insert()
	return err
}
