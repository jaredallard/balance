package account

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

type PlatformName string

const (
	PlatformTelegram PlatformName = "telegram"
)

var (
	ErrUserExists error = errors.New("User already exists")
)

// User is an account owner
type User struct {
	// Id of the User
	Id uuid.UUID `json:"id" pg:",pk,type:uuid,default:uuid_generate_v4()"`

	// PlatformIds are IDs on various platforms this user belongs in
	PlatformIds       map[PlatformName]string `pg:"platform_ids,notnull" json:"platform_ids"`
	PlatformUsernames map[PlatformName]string `pg:"platform_usernames,notnull" json:"platform_usernames"`

	CreatedAt time.Time `json:"created_at" pg:"default:now(),notnull"`
	UpdatedAt time.Time `json:"updated_at" pg:"default:now(),notnull"`
}

func (u *User) String() string {
	return fmt.Sprintf("User<ID: %s, PlatformIds: %v>", u.Id, u.PlatformIds)
}

// FindUser finds a user by their social ID
func (c *Client) FindUser(p PlatformName, id string) (*User, error) {
	u := &User{}
	err := c.db.Model(u).
		Where("platform_ids->>? = ?", p, id).
		Limit(1).
		Select(u)

	return u, err
}

// FindUserByUsername finds a user by their social ID
func (c *Client) FindUserByUsernam(p PlatformName, username string) (*User, error) {
	u := &User{}
	err := c.db.Model(u).
		Where("platform_usernames->>? = ?", p, username).
		Limit(1).
		Select(u)

	return u, err
}

// GetUser returns a user
func (c *Client) GetUser(id uuid.UUID) (*User, error) {
	u := &User{}
	err := c.db.Model(u).
		Where("user.id = ?", u.Id).
		Select()

	return u, err
}

// CreateUser creates a user in the database
func (c *Client) CreateUser(u *User) error {
	// TODO(jaredallard): this is not optimal
	for p, v := range u.PlatformIds {
		if u, err := c.FindUser(p, v); err == nil && u != nil {
			return ErrUserExists
		}
	}
	_, err := c.db.Model(u).Insert()
	return err
}
