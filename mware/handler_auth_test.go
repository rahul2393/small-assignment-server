package mware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gopkg.in/gorp.v1"

	"github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/rahul2393/small-assignment-server/models/acct"
	"github.com/rahul2393/small-assignment-server/testhelpers"
)

const (
	testURL = "/foo?auth-email=test@gmail.com&&auth-token=%s"
)

func TestUserAuthUnauthorized(t *testing.T) {
	db := testhelpers.SetupTestWithFixtures()
	authHandler := TestUserAuth(db)

	req, err := http.NewRequest("GET", "/foo", nil)
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	authHandler.ServeHTTP(rec, req, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "please provide")
}

func TestUserAuthExpiredToken(t *testing.T) {
	db := testhelpers.SetupTestWithFixtures()

	user := getTestUser(t, db)
	token := &acct.Token{UserID: user.ID}
	assert.NoError(t, db.Insert(token))

	token.Expiration = 1
	_, err := db.Update(token)
	assert.NoError(t, err)
	assert.True(t, token.Expired())

	rec := doRequest(t, token, db)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func doRequest(t *testing.T, token *acct.Token, db *gorp.DbMap) *httptest.ResponseRecorder {
	authHandler := TestUserAuth(db)
	u := fmt.Sprintf(testURL, token.String())
	req, err := http.NewRequest("GET", u, nil)
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	authHandler.ServeHTTP(rec, req, func(w http.ResponseWriter, r *http.Request) {})
	return rec
}

func getTestUser(t *testing.T, db *gorp.DbMap) *acct.User {
	user := &acct.User{}
	querySql, args, err := squirrel.Select("*").
		From(acct.TableNameUser).
		Where(squirrel.Eq{"Email": "rahul.agrawal@hotcocoasoftware.com", "Deleted": false}).ToSql()
	assert.Nil(t, err)
	err = db.SelectOne(user, querySql, args...)
	assert.Nil(t, err)
	return user
}
