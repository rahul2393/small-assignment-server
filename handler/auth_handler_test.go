package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.in/gorp.v1"

	"github.com/Masterminds/squirrel"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/negroni"
	"github.com/rahul2393/small-assignment-server/cache"
	"github.com/rahul2393/small-assignment-server/models/acct"
	"github.com/rahul2393/small-assignment-server/mware"
	"github.com/rahul2393/small-assignment-server/testhelpers"
)

const (
	testLoginUrl           = `http://localhost:8000/login`
	testSignUpUrl          = `http://localhost:8000/signup`
	testSignOutUrl         = `http://localhost:8000/api/signout?auth-email=%s&auth-token=%s`
	testCreateUserUrl      = `http://localhost:8000/api/createUser?auth-email=%s&auth-token=%s`
	testResetPasswordUrl   = `http://localhost:8000/api/user/%d/resetPassword?auth-email=%s&auth-token=%s`
	testUpdateUserGroupUrl = `http://localhost:8000/api/user/%d/updateGroup/%d?auth-email=%s&auth-token=%s`
)

func TestLogin(t *testing.T) {
	db := testhelpers.SetupTestWithFixtures()
	mainRouter := mux.NewRouter().StrictSlash(true)
	user, respCode := loginRequest(t, mainRouter, db, "rahul.agrawal@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusOK)
	assert.True(t, user.ID > 0)

	user, respCode = loginRequest(t, mainRouter, db, "rahul.agrawal@hotcocoasoftware.com", "wrong password")
	assert.Equal(t, respCode, http.StatusUnauthorized)

	user, respCode = loginRequest(t, mainRouter, db, "wrong_email@gmail.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusUnauthorized)
}

func TestSignOut(t *testing.T) {
	db := testhelpers.SetupTestWithFixtures()
	mainRouter := mux.NewRouter().StrictSlash(true)
	r := mainRouter.PathPrefix("/api").Subrouter()
	r.Handle("/signout", &testhelpers.TestHandler{
		T:       t,
		Db:      db,
		Handler: SignOut(),
	}).Methods("GET")
	middleware := negroni.New(
		mware.TestUserAuth(db),
		negroni.Wrap(r),
	)
	user, respCode := loginRequest(t, mainRouter, db, "rahul.agrawal@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusOK)
	assert.True(t, user.ID > 0)

	req, err := http.NewRequest("GET", fmt.Sprintf(testSignOutUrl, user.Email, user.Token), nil)
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var userTokens []*acct.Token
	querySql, args, err := squirrel.Select("*").
		From(acct.TableNameToken).
		Where(squirrel.Eq{"UserID": user.ID}).ToSql()
	assert.Nil(t, err)
	_, err = db.Select(&userTokens, querySql, args...)
	assert.Nil(t, err)
	for _, token := range userTokens {
		assert.Equal(t, true, token.Deleted)
	}
}

func TestResetPassword(t *testing.T) {
	db := testhelpers.SetupTestWithFixtures()
	mainRouter := mux.NewRouter().StrictSlash(true)
	r := mainRouter.PathPrefix("/api").Subrouter()
	r.Handle("/user/{id}/resetPassword", &testhelpers.TestHandler{
		T:       t,
		Db:      db,
		Handler: ResetPassword(),
	}).Methods("POST")
	middleware := negroni.New(
		mware.TestUserAuth(db),
		negroni.Wrap(r),
	)

	type form struct {
		OldPassword string
		Password    string
	}
	postBody := &form{OldPassword: "wrong", Password: "updated password"}
	payload, err := json.Marshal(postBody)
	assert.NoError(t, err)

	regularUser, respCode := loginRequest(t, mainRouter, db, "ritik.rishu@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusOK)
	assert.True(t, regularUser.ID > 0)

	req, err := http.NewRequest("POST", fmt.Sprintf(testResetPasswordUrl,
		regularUser.ID, regularUser.Email, regularUser.Token),
		bytes.NewReader(payload))
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	postBody.OldPassword = "i am rahul"
	payload, err = json.Marshal(postBody)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", fmt.Sprintf(testResetPasswordUrl,
		regularUser.ID, regularUser.Email, regularUser.Token),
		bytes.NewReader(payload))
	assert.NoError(t, err)
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	// all the tokens of the user whose password got reset will be deleted
	var userActiveTokens []*acct.Token
	querySql, args, err := squirrel.Select("*").
		From(acct.TableNameToken).
		Where(squirrel.Eq{"UserID": regularUser.ID}).ToSql()
	assert.Nil(t, err)
	_, err = db.Select(&userActiveTokens, querySql, args...)
	assert.Nil(t, err)
	assert.True(t, len(userActiveTokens) == 0)
	// all the cache session of the user will be deleted
	for key := range cache.GetAll() {
		assert.Equal(t, false, strings.HasPrefix(key, regularUser.Email))
	}
	// login with old password will throw 401
	regularUser, respCode = loginRequest(t, mainRouter, db, "ritik.rishu@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusUnauthorized)

	// lower permission level user cannot reset higher level user password
	regularUser, respCode = loginRequest(t, mainRouter, db, "ritik.rishu@hotcocoasoftware.com", "updated password")
	assert.Equal(t, respCode, http.StatusOK)
	assert.True(t, regularUser.ID > 0)
	req, err = http.NewRequest("POST", fmt.Sprintf(testResetPasswordUrl,
		2, regularUser.Email, regularUser.Token),
		bytes.NewReader(payload))
	assert.NoError(t, err)
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSignUp(t *testing.T) {
	db := testhelpers.SetupTestWithFixtures()
	mainRouter := mux.NewRouter().StrictSlash(true)
	mainRouter.Handle("/signup", &testhelpers.TestHandler{
		T:       t,
		Db:      db,
		Handler: SignUp(),
	}).Methods("POST")

	type form struct {
		Email    string
		Name     string
		Password string
	}
	postBody := &form{Email: "test@gmail.com", Name: "test", Password: "test"}
	payload, err := json.Marshal(postBody)
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", testSignUpUrl, bytes.NewReader(payload))
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	mainRouter.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	user := &acct.User{}
	err = json.NewDecoder(rec.Body).Decode(user)
	assert.NoError(t, err)
	assert.True(t, user.ID > 0)
	assert.Equal(t, "test@gmail.com", user.Email)
	assert.Equal(t, "test", user.Name)
	assert.Equal(t, acct.Regular.ID, user.Group.ID)
}

func TestCreateUser(t *testing.T) {
	db := testhelpers.SetupTestWithFixtures()
	mainRouter := mux.NewRouter().StrictSlash(true)
	middleware := negroni.New(
		mware.TestUserAuth(db),
		negroni.Wrap(&testhelpers.TestHandler{
			T:       t,
			Db:      db,
			Handler: CreateUser(),
		}),
	)

	type form struct {
		Email   string
		Name    string
		GroupId int64
	}
	postBody := &form{Email: "test@gmail.com", Name: "test", GroupId: acct.Regular.ID}
	payload, err := json.Marshal(postBody)
	assert.NoError(t, err)

	// case 1: regular user cannot create user
	regularUser, respCode := loginRequest(t, mainRouter, db, "ritik.rishu@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusOK)
	req, err := http.NewRequest("POST", fmt.Sprintf(testCreateUserUrl, regularUser.Email, regularUser.Token),
		bytes.NewReader(payload))
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// case 2: uaer manager can not create admin user
	postBody.GroupId = acct.Admin.ID
	payload, err = json.Marshal(postBody)
	assert.NoError(t, err)
	userManagerUser, respCode := loginRequest(t, mainRouter, db, "rahul.yadav@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusOK)
	req, err = http.NewRequest("POST", fmt.Sprintf(testCreateUserUrl, userManagerUser.Email, userManagerUser.Token),
		bytes.NewReader(payload))
	assert.NoError(t, err)
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// case 3: Invalid group Id cannot be created
	postBody.GroupId = 0
	payload, err = json.Marshal(postBody)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", fmt.Sprintf(testCreateUserUrl, userManagerUser.Email, userManagerUser.Token),
		bytes.NewReader(payload))
	assert.NoError(t, err)
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// case 4: UserManger can create another user manager
	postBody.GroupId = acct.UserManager.ID
	payload, err = json.Marshal(postBody)
	assert.NoError(t, err)
	req, err = http.NewRequest("POST", fmt.Sprintf(testCreateUserUrl, userManagerUser.Email, userManagerUser.Token),
		bytes.NewReader(payload))
	assert.NoError(t, err)
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	user := &acct.User{}
	err = json.NewDecoder(rec.Body).Decode(user)
	assert.NoError(t, err)
	assert.True(t, user.ID > 0)
	assert.Equal(t, "test@gmail.com", user.Email)
	assert.Equal(t, "test", user.Name)
	assert.Equal(t, acct.UserManager.ID, user.Group.ID)
}

func TestUpdateUserGroup(t *testing.T) {
	db := testhelpers.SetupTestWithFixtures()
	mainRouter := mux.NewRouter().StrictSlash(true)
	r := mainRouter.PathPrefix("/api").Subrouter()
	r.Handle("/user/{id}/updateGroup/{groupId}", &testhelpers.TestHandler{
		T:       t,
		Db:      db,
		Handler: UpdateUserGroup(),
	}).Methods("GET")
	middleware := negroni.New(
		mware.TestUserAuth(db),
		negroni.Wrap(r),
	)

	// case 1: regular user cannot update group of regular user
	regularUser, respCode := loginRequest(t, mainRouter, db, "ritik.rishu@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusOK)
	req, err := http.NewRequest("GET", fmt.Sprintf(testUpdateUserGroupUrl, regularUser.ID, acct.UserManager.ID,
		regularUser.Email, regularUser.Token),
		nil)
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// case 2: uaer manager can not update and user to admin user
	userManager, respCode := loginRequest(t, mainRouter, db, "rahul.yadav@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusOK)
	req, err = http.NewRequest("GET", fmt.Sprintf(testUpdateUserGroupUrl, regularUser.ID, acct.Admin.ID,
		userManager.Email, userManager.Token),
		nil)
	assert.NoError(t, err)
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// case 3: uaer manager can update user to user manager
	userManager, respCode = loginRequest(t, mainRouter, db, "rahul.yadav@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusOK)
	req, err = http.NewRequest("GET", fmt.Sprintf(testUpdateUserGroupUrl, userManager.ID, acct.UserManager.ID,
		userManager.Email, userManager.Token),
		nil)
	assert.NoError(t, err)
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// case 4: Admin can update user to any group
	adminUser, respCode := loginRequest(t, mainRouter, db, "rahul.agrawal@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, respCode, http.StatusOK)
	req, err = http.NewRequest("GET", fmt.Sprintf(testUpdateUserGroupUrl, regularUser.ID, acct.Admin.ID,
		adminUser.Email, adminUser.Token),
		nil)
	assert.NoError(t, err)
	rec = httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	regularUser, respCode = loginRequest(t, mainRouter, db, "ritik.rishu@hotcocoasoftware.com", "i am rahul")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, acct.Admin.ID, regularUser.Group.ID)
}

func loginRequest(t *testing.T, mainRouter *mux.Router, db *gorp.DbMap, email, password string) (*acct.User, int) {
	mainRouter.Handle("/login", &testhelpers.TestHandler{
		T:       t,
		Db:      db,
		Handler: Login(),
	}).Methods("POST")

	type form struct {
		Email    string
		Password string
	}

	postBody := &form{Email: email, Password: password}
	payload, err := json.Marshal(postBody)
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", testLoginUrl, bytes.NewReader(payload))
	assert.NoError(t, err)
	rec := httptest.NewRecorder()
	mainRouter.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		return nil, rec.Code
	}
	user := &acct.User{}
	err = json.NewDecoder(rec.Body).Decode(user)
	assert.NoError(t, err)
	return user, rec.Code
}
