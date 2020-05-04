package handler

import (
	//"fmt"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gopkg.in/gorp.v1"

	"github.com/Masterminds/squirrel"
	"github.com/dchest/uniuri"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rahul2393/small-assignment-server/httperr"
	"github.com/rahul2393/small-assignment-server/logger"
	"github.com/rahul2393/small-assignment-server/models/acct"
	"github.com/rahul2393/small-assignment-server/mware"
)

const (
	loginErrorMessage = "Incorrect email and password combination."
)

func SignOut() mware.Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		v := r.URL.Query()
		token := v.Get("auth-token")
		currentUser := acct.GetCurrentRequestUserFromCache(r.Header.Get("X-Request-Id"))
		_, id, err := acct.SplitToken(token)
		if err != nil {
			return httperr.New(http.StatusBadRequest, "invalid token", err)
		}
		// retrieve and validate token
		t := &acct.Token{}
		query, args, _ := squirrel.Select("*").
			From(acct.TableNameToken).
			Where(squirrel.Eq{"ID": id, "UserID": currentUser.ID}).ToSql()
		if err := db.SelectOne(t, query, args...); err != nil {
			return httperr.New(http.StatusBadRequest, "invalid token", err)
		}
		t.Deleted = true
		if _, err := db.Update(t); err != nil {
			return httperr.New(http.StatusBadRequest, "signout failed", err)
		}
		return nil
	}
}

func SignUp() mware.Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		logger.Debug("inside signUp")
		trans, err := db.Begin()
		if err != nil {
			return httperr.NewInternal(err)
		}
		defer func() {
			if err != nil {
				trans.Rollback()
			} else {
				trans.Commit()
			}
		}()
		// parse form
		type Form struct {
			Email                  string
			Name                   string
			Password               string
			ExpectedCaloriesPerDay int64
		}
		form := &Form{}
		if err = json.NewDecoder(r.Body).Decode(form); err != nil {
			logger.ErrorWithMsg("problem decoding form", err)
			return httperr.New(http.StatusUnauthorized, "invalid request", errors.Wrap(err, "cannot decode"))
		}

		newUser := acct.User{Email: form.Email,
			Name: form.Name,
			ExpectedCaloriesPerDay: form.ExpectedCaloriesPerDay}
		newUser.SetPassword(form.Password)

		// set the account of newly created user to be regular account
		newUser.GroupID = acct.Regular.ID
		if err = trans.Insert(&newUser); err != nil {
			logger.Error(err)
			return httperr.New(http.StatusUnauthorized, "invalid request", err)
		}
		token := &acct.Token{
			UserID: newUser.ID,
		}
		if err = trans.Insert(token); err != nil {
			logger.Debug("problem inserting token")
			return httperr.New(http.StatusUnauthorized, "error in inserting token", err)
		}
		// return user w/ token
		newUser.Token = token.String()
		newUser.TokenExpiration = token.Expiration
		if err = newUser.Expand(trans, ""); err != nil {
			logger.Debugf("problem expanding user")
			return httperr.New(http.StatusUnauthorized, "error in expanding user", err)
		}
		return json.NewEncoder(w).Encode(newUser)
	}
}

func ResetPassword() mware.Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		params := mux.Vars(r)
		type Form struct {
			OldPassword string
			Password    string
		}
		form := &Form{}
		if err := json.NewDecoder(r.Body).Decode(form); err != nil {
			return httperr.New(http.StatusBadRequest, "invalid request", err)
		}

		currentUser := acct.GetCurrentRequestUserFromCache(r.Header.Get("X-Request-Id"))

		trans, err := db.Begin()
		if err != nil {
			return httperr.NewInternal(err)
		}
		defer func() {
			if err != nil {
				trans.Rollback()
			} else {
				trans.Commit()
			}
		}()

		fetchUserQuery, args, err := squirrel.Select("*").From(acct.TableNameUser).Where(squirrel.Eq{"ID": params["id"],
			"Deleted": 0}).ToSql()
		if err != nil {
			return httperr.New(http.StatusBadRequest, "user not active in system", err)
		}
		userToUpdate := &acct.User{}
		if err = trans.SelectOne(userToUpdate, fetchUserQuery, args...); err != nil {
			return httperr.New(http.StatusBadRequest, "user not active in system", err)
		}

		if !userToUpdate.HasPassword(form.OldPassword) {
			return httperr.New(http.StatusBadRequest, "invalid old password", err)
		}

		userToUpdate.Expand(trans, "")
		if currentUser.ID != userToUpdate.ID &&
			!canResetPassword(currentUser.Group.ID, userToUpdate.GroupID) {
			err = fmt.Errorf("handler: invite user invalid request")
			return httperr.New(
				http.StatusBadRequest,
				"user do not have permission to perform request",
				err)
		}

		if err = userToUpdate.SetPassword(form.Password); err != nil {
			return httperr.New(
				http.StatusBadRequest,
				"problem in resetting user password",
				err)
		}

		if _, err = trans.Update(userToUpdate); err != nil {
			return httperr.New(
				http.StatusBadRequest,
				"problem in resetting user password",
				err)
		}

		// delete session from cache
		userToUpdate.DeleteCacheSession()

		// delete all the tokens once password is reset
		if _, err = trans.Exec(fmt.Sprintf("Delete from %s where userID = %d",
			acct.TableNameToken, userToUpdate.ID)); err != nil {
			return httperr.New(
				http.StatusBadRequest,
				"problem in resetting user password",
				err)
		}
		return json.NewEncoder(w).Encode(userToUpdate)
	}
}

func Login() mware.Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		// parse form
		type Form struct {
			Email    string
			Password string
		}
		logger.Debug("inside login")
		form := &Form{}
		if err := json.NewDecoder(r.Body).Decode(form); err != nil {
			return httperr.New(http.StatusUnauthorized, "invalid request", err)
		}
		form.Email = strings.ToLower(form.Email)
		// find user
		loginUser := &acct.User{}
		query, args, _ := squirrel.Select("*").
			From(acct.TableNameUser).
			Where(squirrel.Eq{"Email": form.Email, "Deleted": false}).ToSql()
		if err := db.SelectOne(loginUser, query, args...); err != nil {
			logger.Debugf("couldn't find user w/ email %s %s", form.Email, err)
			return httperr.New(http.StatusUnauthorized, "email doesn't exist", err)
		}
		// check password
		if !loginUser.HasPassword(form.Password) {
			logger.Debugf("problem checking password")
			return httperr.New(http.StatusUnauthorized, loginErrorMessage, errors.New(loginErrorMessage))
		}
		//// create token
		token := &acct.Token{
			UserID: loginUser.ID,
		}
		if err := db.Insert(token); err != nil {
			logger.Debugf("problem inserting token")
			return errors.Wrap(err, "error in inserting token")
		}
		// return user w/ token
		loginUser.Token = token.String()
		loginUser.TokenExpiration = token.Expiration
		if err := loginUser.Expand(db, ""); err != nil {
			logger.Debugf("problem expanding user")
			return errors.Wrap(err, "error in expanding user")
		}
		return json.NewEncoder(w).Encode(loginUser)
	}
}

func CreateUser() mware.Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		type Form struct {
			Email                  string
			Name                   string
			ExpectedCaloriesPerDay int64
			GroupID                int64
		}
		form := &Form{}
		if err := json.NewDecoder(r.Body).Decode(form); err != nil {
			err = fmt.Errorf("handler: create user invalid json  - %v", err)
			return httperr.New(http.StatusBadRequest, "invalid request", err)
		}
		form.Email = strings.ToLower(form.Email)

		group := acct.GroupForID(form.GroupID)
		if group == nil {
			err := fmt.Errorf("handler: must specify group")
			return httperr.New(http.StatusBadRequest, "invalid request", err)
		}
		currentUser := acct.GetCurrentRequestUserFromCache(r.Header.Get("X-Request-Id"))
		if !canModifyGroup(currentUser.Group.ID, form.GroupID, form.GroupID) {
			return httperr.New(
				http.StatusBadRequest,
				"user do not have permission to perform request",
				fmt.Errorf("handler: invite user invalid request"))
		}
		trans, err := db.Begin()
		if err != nil {
			return httperr.NewInternal(err)
		}
		defer func() {
			if err != nil {
				trans.Rollback()
			} else {
				trans.Commit()
			}
		}()
		userToCreate := acct.User{Email: form.Email,
			Name: form.Name,
			ExpectedCaloriesPerDay: form.ExpectedCaloriesPerDay}
		p := uniuri.NewLen(8)
		userToCreate.SetPassword(p)
		// set the account of newly created user to be regular account
		userToCreate.GroupID = form.GroupID
		if err = trans.Insert(&userToCreate); err != nil {
			logger.Error(err)
			return httperr.New(http.StatusUnauthorized, "invalid request", err)
		}
		token := &acct.Token{
			UserID: userToCreate.ID,
		}
		if err = trans.Insert(token); err != nil {
			logger.Debug("problem inserting token")
			return httperr.New(http.StatusUnauthorized, "error in inserting token", err)
		}
		// return user w/ token
		userToCreate.Token = token.String()
		userToCreate.TokenExpiration = token.Expiration
		if err = userToCreate.Expand(trans, ""); err != nil {
			logger.Debugf("problem expanding user")
			return httperr.New(http.StatusUnauthorized, "error in expanding user", err)
		}
		return json.NewEncoder(w).Encode(userToCreate)
	}
}

func UpdateUserGroup() mware.Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		params := mux.Vars(r)
		userToUpdate := &acct.User{}
		query, args, _ := squirrel.Select("*").
			From(acct.TableNameUser).
			Where(squirrel.Eq{"ID": params["id"], "Deleted": false}).ToSql()
		if err := db.SelectOne(userToUpdate, query, args...); err != nil {
			logger.Debugf("couldn't find user w/ ID %d %s", params["id"], err)
			return httperr.New(http.StatusBadRequest, "user doesn't exist", err)
		}

		groupId, err := strconv.ParseInt(params["groupId"], 10, 64)
		if err != nil {
			logger.Debugf("invalid group id %d", groupId, err)
			return httperr.New(http.StatusBadRequest, "invalid group", err)
		}

		group := acct.GroupForID(groupId)
		if group == nil {
			err := fmt.Errorf("handler: must specify group")
			return httperr.New(http.StatusBadRequest, "invalid request", err)
		}
		currentUser := acct.GetCurrentRequestUserFromCache(r.Header.Get("X-Request-Id"))
		if !canModifyGroup(currentUser.Group.ID, userToUpdate.GroupID, groupId) {
			return httperr.New(
				http.StatusBadRequest,
				"user do not have permission to perform request",
				fmt.Errorf("handler: invite user invalid request"))
		}
		trans, err := db.Begin()
		if err != nil {
			return httperr.NewInternal(err)
		}
		defer func() {
			if err != nil {
				trans.Rollback()
			} else {
				trans.Commit()
			}
		}()
		userToUpdate.GroupID = groupId
		if _, err = trans.Update(userToUpdate); err != nil {
			logger.Debug("problem updating user group")
			return httperr.New(http.StatusUnauthorized, "error in updating user group", err)
		}
		if err = userToUpdate.Expand(trans, ""); err != nil {
			logger.Debugf("problem expanding user")
			return httperr.New(http.StatusUnauthorized, "error in expanding user", err)
		}

		// delete session from cache
		userToUpdate.DeleteCacheSession()

		// delete tokens from cache
		if _, err = trans.Exec(fmt.Sprintf("Delete from %s where userID = %d",
			acct.TableNameToken, userToUpdate.ID)); err != nil {
			return httperr.New(
				http.StatusBadRequest,
				"problem in resetting user password",
				err)
		}
		return json.NewEncoder(w).Encode(userToUpdate)
	}
}

func canModifyGroup(currentUserGroupID, otherUserGroupID int64, finalGroup int64) bool {
	switch currentUserGroupID {
	case acct.Admin.ID:
		return true
	case acct.UserManager.ID:
		if otherUserGroupID == acct.Admin.ID {
			return false
		}
		if finalGroup > 0 && finalGroup == acct.Admin.ID {
			return false
		}
		return true
	default:
		return false
	}
	return false
}

func canResetPassword(currentUserGroupID, otherUserGroupID int64) bool {
	switch currentUserGroupID {
	case acct.Admin.ID:
		return true
	case acct.UserManager.ID:
		if otherUserGroupID != acct.Regular.ID {
			return false
		}
		return true
	default:
		return false
	}
	return false
}
