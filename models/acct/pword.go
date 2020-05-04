package acct

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

func hash(password string) (hash string, err error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.New("error in generating hash of password")
	}
	return string(b), nil
}

func validPassword(password, hash string) bool {
	return nil == bcrypt.CompareHashAndPassword(
		[]byte(hash), []byte(password))
}
