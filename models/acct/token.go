package acct

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/gorp.v1"

	"github.com/ShaleApps/gator"
	"github.com/dchest/uniuri"
	"github.com/pkg/errors"
	"github.com/rahul2393/small-assignment-server/milli"
	"github.com/rahul2393/small-assignment-server/models/basemodel"
)

const (
	// ModelNameToken is the name of the token model.
	ModelNameToken = "Token"
	// TableNameToken is the name of the token sql table.
	TableNameToken = "tokens"
)

// Token represents an authentication token for a user
type Token struct {
	basemodel.BaseModel

	UserID     int64  `gator:"nonzero"`
	Value      string `db:"-"`
	Hash       string `gator:"nonzero"`
	Expiration int64  `gator:"nonzero"`
}

func (t *Token) String() string {
	return fmt.Sprintf("%s:%d", t.Value, t.ID)
}

// Expired returns whether or not the token is expired.
func (t *Token) Expired() bool {
	exp := milli.Time(t.Expiration)
	return time.Now().After(exp)
}

// HasValue returns whether or not the value is equal
// to the hashed value.
func (t *Token) HasValue(value string) bool {
	return validPassword(value, t.Hash)
}

func (t *Token) PreInsert(s gorp.SqlExecutor) error {
	t.Created = milli.Timestamp(time.Now())
	t.Updated = milli.Timestamp(time.Now())
	value := uniuri.NewLen(15)
	hash, err := hash(value)
	if err != nil {
		return errors.New("Error in creating token hash value")
	}
	t.Value = value
	t.Hash = hash
	ex := time.Now().AddDate(0, 0, 7)
	t.Expiration = milli.Timestamp(ex)
	if err := gator.NewStruct(t).Validate(); err != nil {
		return errors.Wrap(err, "error in validating access-token")
	}
	return nil
}

// PostInsert implements the gorp.HasPostInsert interface.
func (t *Token) PostInsert(s gorp.SqlExecutor) error {
	timestamp := milli.Timestamp(time.Now())
	if _, err := s.Exec("delete from "+TableNameToken+" where Expiration < ?", timestamp); err != nil {
		return errors.New("Error in deleting expired access-token")
	}
	return nil
}

func (t *Token) TableName() string {
	return TableNameToken
}

// SplitToken seperates the token encoded in the query string into its value and id.
// An error is returned the token format is invalid.
func SplitToken(token string) (value, id string, err error) {
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return "", "", errors.New("account: invalid token format")
	}
	return parts[0], parts[1], nil
}
