package acct

import (
	"net/http"
	"strings"
	"time"

	"gopkg.in/gorp.v1"

	"github.com/Masterminds/squirrel"
	"github.com/ShaleApps/gator"
	"github.com/pkg/errors"
	"github.com/rahul2393/small-assignment-server/cache"
	"github.com/rahul2393/small-assignment-server/httperr"
	"github.com/rahul2393/small-assignment-server/milli"
	"github.com/rahul2393/small-assignment-server/models/basemodel"
)

const (
	// ModelNameUser is the name of the user model.
	ModelNameUser = "User"
)

type Verifier interface {
	Verify(reqId string, s gorp.SqlExecutor) error
}

type User struct {
	basemodel.BaseModel

	Email                  string `json:"email"`
	Name                   string `json:"name"`
	Password               string `db:"-" json:"password,omitempty"`
	ExpectedCaloriesPerDay int64  `json:"expectedCaloriesPerDay"`

	Token        string `db:"-" json:"token,omitempty"`
	PasswordHash string `json:"-"`

	TokenExpiration int64  `db:"-" json:"tokenExpiration"`
	GroupID         int64  `db:"groupID" json:"-"`
	Group           *Group `db:"-" json:"group,omitempty"`
}

func (u *User) Merge(src interface{}) error {
	from, ok := src.(*User)
	if !ok {
		return ErrMergeWrongType
	}

	u.Name = from.Name
	u.Email = from.Email
	if from.Password != "" {
		u.SetPassword(from.Password)
	}
	u.ExpectedCaloriesPerDay = from.ExpectedCaloriesPerDay
	return nil
}

func (u *User) TableName() string {
	return TableNameUser
}

// HasPassword returns whether or not the user has the given password.
func (u *User) HasPassword(password string) bool {
	return validPassword(password, u.PasswordHash)
}

func (u *User) PreInsert(s gorp.SqlExecutor) error {
	u.Created = milli.Timestamp(time.Now())
	u.Updated = milli.Timestamp(time.Now())
	u.Email = strings.ToLower(u.Email)
	if u.Password == "" {
		return errors.New("auth: user requires password")
	}
	if err := gator.NewStruct(u).Validate(); err != nil {
		return errors.Wrap(err, "error in validating user")
	}
	return nil
}

func (u *User) PostUpdate(s gorp.SqlExecutor) error {
	//update cache
	for key := range cache.GetAll() {
		cacheKeys := strings.Split(key, ":")
		cacheEmail := cacheKeys[0]
		cacheToken := cacheKeys[1]
		if len(cacheKeys) == 3 {
			cacheToken = cacheToken + ":" + cacheKeys[2]
		}
		if cacheEmail == u.Email {
			cache.Set(cacheEmail, cacheToken, cache.Item{Src: u})
		}
	}
	return nil
}

func (u *User) Delete(s gorp.SqlExecutor) error {
	u.Deleted = true
	return nil
}

func (u *User) PreUpdate(s gorp.SqlExecutor) error {
	u.Updated = milli.Timestamp(time.Now())
	if err := gator.NewStruct(u).Validate(); err != nil {
		return errors.Wrap(err, "error in validating user")
	}
	return nil
}

// SetPassword sets the password and password hash.
func (u *User) SetPassword(p string) error {
	h, err := hash(p)
	if err != nil {
		return errors.Wrap(err, "error in creating hash of password")
	}
	u.Password = p
	u.PasswordHash = h
	return nil
}

func (u *User) Expand(s gorp.SqlExecutor, exclude string) error {
	u.ModelName = ModelNameUser
	u.Group = GroupForID(u.GroupID)
	return nil
}

func errInvalidAuth(err error) error {
	return httperr.New(
		http.StatusUnauthorized,
		"Invalid Authentication Credentials",
		err)
}

func Authenticate(
	s gorp.SqlExecutor,
	email, token string) (*User, error) {
	// check if credentials are provided
	email = strings.ToLower(email)
	if email == "" || token == "" {
		return nil, errInvalidAuth(errors.New("account: authentication credentials missing"))
	}

	if src, in := cache.Get(email, token); in {
		return src.(*User), nil
	}
	// handleErr filters 401 or 500 error
	handleErr := func(err error) (*User, error) {
		return nil, errInvalidAuth(err)
	}
	user := &User{}
	query, args, _ := squirrel.Select("*").
		From(TableNameUser).
		Where(squirrel.Eq{"Email": email}).ToSql()
	if err := s.SelectOne(user, query, args...); err != nil {
		return handleErr(err)
	}
	// retrieve token from cache or db
	value, id, err := SplitToken(token)
	if err != nil {
		return nil, errInvalidAuth(err)
	}
	// retrieve and validate token
	t := &Token{}
	query, args, _ = squirrel.Select("*").
		From(TableNameToken).
		Where(squirrel.Eq{"ID": id}).ToSql()
	if err := s.SelectOne(t, query, args...); err != nil {
		return handleErr(err)
	}
	if t.Expired() {
		return handleErr(errors.New("acct: token is expired"))
	}
	if !t.HasValue(value) {
		return handleErr(errors.New("acct: token's value is incorrect"))
	}
	if err := user.Expand(s, ""); err != nil {
		return nil, errors.Wrap(err, "acct: error in expanding user")
	}
	item := cache.Item{Src: user, Duration: time.Hour}
	cache.Set(email, token, item)
	return user, nil
}

func (u *User) DeleteCacheSession() {
	for key := range cache.GetAll() {
		cacheKeys := strings.Split(key, ":")
		if len(cacheKeys) == 3 {
			cacheEmail := cacheKeys[0]
			cacheToken := cacheKeys[1] + ":" + cacheKeys[2]
			if cacheEmail == u.Email {
				cache.Delete(cacheEmail, cacheToken)
			}
		}
	}
	return
}
