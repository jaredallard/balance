package account

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// Transaction is a user transaction
type Transaction struct {
	// Id of the User
	Id uuid.UUID `json:"id" pg:",pk,type:uuid,default:uuid_generate_v4()"`

	// CreatedBy is the user who created this transaction
	CreatedBy uuid.UUID `pg:"created_by,type:uuid" json:"created_by"`

	// Involved are the users that were involved in this transaction
	Involved []uuid.UUID `pg:"involved" json:"involved"`

	// Amount that this transaction was for, split across all involved users
	Amount float64 `json:"amount" pg:"amount"`

	CreatedAt time.Time `json:"created_at" pg:"default:now(),notnull"`
}

func (t *Transaction) String() string {
	return fmt.Sprintf("Transaction<ID: %s, Amount: %v, CreatedBy: %s, Involved: %s>", t.Id, t.Amount, t.CreatedBy, t.Involved)
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
	involvedIDs := make([]uuid.UUID, len(involved))
	for i, u := range involved {
		involvedIDs[i] = u.Id
	}

	t := Transaction{
		CreatedBy: createdBy.Id,
		Involved:  involvedIDs,
		Amount:    amount,
		CreatedAt: time.Now(),
	}

	_, err := c.db.Model(t).Insert()
	return err
}
