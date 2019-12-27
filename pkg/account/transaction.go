package account

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"
)

// Transaction is a user transaction
type Transaction struct {
	// Id of the User
	Id uuid.UUID `json:"id" pg:",pk,type:uuid,default:uuid_generate_v4()"`

	// CreatedBy is the user who created this transaction
	CreatedBy uuid.UUID `pg:"created_by,type:uuid" json:"created_by"`

	// Account is a userID -> accountID mapping of accounts that
	// were hit during this transaction
	Accounts map[uuid.UUID]uuid.UUID `pg:"accounts" json:"account_id"`

	// Amount that this transaction was for, split across all involved users
	Amount float64 `json:"amount" pg:"amount"`

	CreatedAt time.Time `json:"created_at" pg:"default:now(),notnull"`
}

func (t *Transaction) String() string {
	return fmt.Sprintf("Transaction<ID: %s, Amount: %v, CreatedBy: %s, Accounts: %s>", t.Id, t.Amount, t.CreatedBy, t.Accounts)
}

func (c *Client) GetAllTransactionsByUser(u *User, filterUser *User) ([]*Transaction, error) {
	trans := []*Transaction{}
	query := c.db.Model(&trans).
		Where("accounts->>? != '' OR created_by = ?", u.Id, u.Id)

	// append a filter for the user we want
	if filterUser != nil {
		query = query.Where("accounts->>? != '' OR created_by = ?", filterUser.Id, filterUser.Id)
	}

	err := query.Select()

	return trans, err
}

// GetTransaction by ID
func (c *Client) GetTransaction(id string) (*Transaction, error) {
	t := &Transaction{}
	err := c.db.Model(t).
		Where("transaction.id = ?", id).
		Limit(1).
		Select(t)

	return t, err
}

// CreateUser creates a new transaction in the database
func (c *Client) CreateTransaction(createdBy *User, involved []User, amount float64) error {
	accounts := make(map[uuid.UUID]uuid.UUID)
	for _, u := range involved {
		a, err := c.FindAccountBetween(createdBy, &u)
		if err != nil {
			log.Errorf("failed to get accounts")
			continue
		}
		accounts[u.Id] = a.Id
	}

	t := &Transaction{
		CreatedBy: createdBy.Id,
		Accounts:  accounts,
		Amount:    amount,
		CreatedAt: time.Now(),
	}

	_, err := c.db.Model(t).Insert()
	return err
}
