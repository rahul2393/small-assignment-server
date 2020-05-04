package mware

import (
	"fmt"
	"gopkg.in/gorp.v1"
	"net/http"

	"github.com/satori/go.uuid"
	"github.com/rahul2393/small-assignment-server/dbutil"
	"github.com/rahul2393/small-assignment-server/httperr"
	"github.com/rahul2393/small-assignment-server/logger"
	"github.com/rahul2393/small-assignment-server/models/acct"
)

const (
	authKeyEmail = "auth-email"
	authKeyToken = "auth-token"

	requestHeader = "X-Request-Id"
)

type Handler func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dbMap, err := dbutil.DB()
	if err != nil {
		logger.Error(err)
		httperr.Write(w, err)
		return
	}
	// There is an issue where the number of connections either maxes out or expires.
	// This is an effort to keep that from happening.
	if err = dbMap.Db.Ping(); err != nil {
		logger.Debugf("The database could not be hit: %s", err)
		logger.Error(err)
		httperr.Write(w, err)
		return
	}
	if err := h(w, r, dbMap); err != nil {
		logger.ErrorMsg(fmt.Sprintf("%+v", err))
		httperr.Write(w, err)
		return
	}
}

// JSONContentType is negroni compatible middleware that writes out the json
// content type header.
type JSONContentType struct{}

// ServeHTTP implements the negroni.Handler interface
func (m JSONContentType) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	w.Header().Add("Content-Type", "application/json")
	next(w, r)
}

type BaseMiddleWare func(http.ResponseWriter, *http.Request, http.HandlerFunc) error

// ServeHTTP implements the negroni.Handler interface
func (m BaseMiddleWare) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if err := m(w, r, next); err != nil {
		httperr.Write(w, err)
		return
	}
}

func UserAuth() BaseMiddleWare {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) error {
		v := r.URL.Query()
		requestId := ""
		if r.Header.Get(requestHeader) == "" {
			reqId, _ := uuid.NewV4()
			requestId = fmt.Sprintf("%s", reqId)
			r.Header.Set(requestHeader, requestId)
		} else {
			requestId = r.Header.Get(requestHeader)
		}
		email := v.Get(authKeyEmail)
		token := v.Get(authKeyToken)
		if email == "" || token == "" {
			err := fmt.Errorf("please provide %s and %s", authKeyEmail, authKeyToken)
			err = httperr.New(http.StatusUnauthorized, "Incomplete details for request", err)
			return err
		}
		logger.Debugf("user auth middleware")
		db, err := dbutil.DB()
		if err != nil {
			return err
		}
		//// authenticate user
		user, err := acct.Authenticate(db, email, token)
		if err != nil {
			return err
		}
		//// setup request mapping
		acct.ReqSetUser(requestId, user)
		next(w, r)
		// clean up request
		acct.ReqDeleteUser(requestId)
		return nil
	}
}

func TestUserAuth(db *gorp.DbMap) BaseMiddleWare {
	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) error {
		v := r.URL.Query()
		requestId := ""
		if r.Header.Get(requestHeader) == "" {
			reqId, _ := uuid.NewV4()
			requestId = fmt.Sprintf("%s", reqId)
			r.Header.Set(requestHeader, requestId)
		} else {
			requestId = r.Header.Get(requestHeader)
		}
		email := v.Get(authKeyEmail)
		token := v.Get(authKeyToken)
		if email == "" || token == "" {
			err := fmt.Errorf("please provide %s and %s", authKeyEmail, authKeyToken)
			err = httperr.New(http.StatusUnauthorized, "Incomplete details for request", err)
			return err
		}
		logger.Debugf("user auth middleware")
		//// authenticate user
		user, err := acct.Authenticate(db, email, token)
		if err != nil {
			return err
		}
		//// setup request mapping
		acct.ReqSetUser(requestId, user)
		next(w, r)
		// clean up request
		acct.ReqDeleteUser(requestId)
		return nil
	}
}
