package mware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"gopkg.in/gorp.v1"

	"github.com/Masterminds/squirrel"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rahul2393/small-assignment-server/httperr"
	"github.com/rahul2393/small-assignment-server/logger"
	"github.com/rahul2393/small-assignment-server/models/acct"
	"github.com/rahul2393/small-assignment-server/models/basemodel"
	"github.com/rahul2393/small-assignment-server/permission"
	"github.com/rahul2393/small-assignment-server/rest"
)

const (
	read = iota + 1
	Write
)

type Merger interface {
	Merge(src interface{}) error
}

type MergeModel interface {
	basemodel.Model
	Merger
}

func GetAll(m basemodel.Model) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		var (
			params = r.URL.Query()
			models []interface{}
			limit  = rest.UintFromKey(params, rest.KeyLimit, rest.LimitDefault)
		)
		user := acct.GetCurrentRequestUserFromCache(r.Header.Get(requestHeader))
		requiredPermission := user.Group.GetPermission(m.TableName(), read)
		if requiredPermission == acct.NoPerm {
			err := errors.New("mware: inadequate permissions for request")
			return httperr.New(http.StatusForbidden, "Inadequate permissions for request.", err)
		}

		q, err := rest.Query(m, m.TableName(), params)
		if err != nil {
			return clientError(err)
		}
		q.Query = q.Query.Limit(limit)
		q.Query = permissions.UserLevelFilter(q.Query, m, user, requiredPermission)

		sql, args, _ := q.Query.ToSql()
		logger.Debug(fmt.Sprintf("sql %s and args %v\n", sql, args))
		fmt.Printf("sql %s and args %v\n", sql, args)
		models, err = db.Select(m, sql, args...)
		if err != nil {
			return clientError(err)
		}
		if params.Get(rest.KeyExpand) == "true" {
			if _, ok := m.(basemodel.Expander); ok {
				logger.Debugf("Expanding")
				for _, m := range models {
					e, _ := m.(basemodel.Expander)
					if err := e.Expand(db, params.Get(rest.KeyExclude)); err != nil {
						return err
					}
				}
			}
		}
		return json.NewEncoder(w).Encode(models)
	}
}

func GetByID(m basemodel.Model) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		user := acct.GetCurrentRequestUserFromCache(r.Header.Get(requestHeader))
		requiredPermission := user.Group.GetPermission(m.TableName(), read)
		if requiredPermission == acct.NoPerm {
			err := errors.New("mware: inadequate permissions for request")
			return httperr.New(http.StatusForbidden, "Inadequate permissions for request.", err)
		}
		values := r.URL.Query()
		params := mux.Vars(r)
		if err := GetID(db, user, m, params["id"], requiredPermission); err != nil {
			return err
		}
		if e, ok := m.(basemodel.Expander); ok {
			if err := e.Expand(db, values.Get(rest.KeyExclude)); err != nil {
				return err
			}
		}
		return json.NewEncoder(w).Encode(m)
	}
}

func Create(m basemodel.Model) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		user := acct.GetCurrentRequestUserFromCache(r.Header.Get(requestHeader))
		requiredPermission := user.Group.GetPermission(m.TableName(), Write)
		if requiredPermission == acct.NoPerm {
			err := errors.New("mware: inadequate permissions for request")
			return httperr.New(http.StatusForbidden, "Inadequate permissions for request.", err)
		}

		mCopy := copyResource(m)
		if err := json.NewDecoder(r.Body).Decode(mCopy); err != nil {
			return clientError(err)
		}
		trans, err := db.Begin()
		if err != nil {
			message := fmt.Sprintf("%s could not begin create transaction.", m.TableName())
			return httperr.New(http.StatusInternalServerError, message, err)
		}

		defer func() {
			if err != nil {
				if err = trans.Rollback(); err != nil {
					logger.ErrorMsgf("unable to rollback %s create transaction: %s", m.TableName(), err.Error())
				}
			} else {
				if err = trans.Commit(); err != nil {
					logger.ErrorMsgf("unable to commit %s create transaction: %s", m.TableName(), err.Error())
				}
			}
		}()

		if verifier, ok := mCopy.(acct.Verifier); ok {
			if err = verifier.Verify(r.Header.Get(requestHeader), trans); err != nil {
				return FullHttpErrorAndDebug(err, m.TableName(), "verifier validation")
			}
		}

		if err = trans.Insert(mCopy); err != nil {
			return clientError(err)
		}
		if e, ok := mCopy.(basemodel.Expander); ok {
			if err = e.Expand(db, ""); err != nil {
				return err
			}
		}
		return json.NewEncoder(w).Encode(mCopy)
	}
}

func UpdateByID(m MergeModel) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		user := acct.GetCurrentRequestUserFromCache(r.Header.Get(requestHeader))
		requiredPermission := user.Group.GetPermission(m.TableName(), Write)
		if requiredPermission == acct.NoPerm {
			err := errors.New("mware: inadequate permissions for request")
			return httperr.New(http.StatusForbidden, "Inadequate permissions for request.", err)
		}
		params := mux.Vars(r)
		mCopy := copyResource(m)
		err := GetID(db, user, mCopy, params["id"], requiredPermission)
		if err != nil {
			return err
		}
		from := copyResource(m)
		if err := json.NewDecoder(r.Body).Decode(from); err != nil {
			return clientError(err)
		}
		trans, err := db.Begin()
		if err != nil {
			message := fmt.Sprintf("%s could not begin create transaction.", m.TableName())
			return httperr.New(http.StatusInternalServerError, message, err)
		}

		defer func() {
			if err != nil {
				if err = trans.Rollback(); err != nil {
					logger.ErrorMsgf("unable to rollback %s create transaction: %s", m.TableName(), err.Error())
				}
			} else {
				if err = trans.Commit(); err != nil {
					logger.ErrorMsgf("unable to commit %s create transaction: %s", m.TableName(), err.Error())
				}
			}
		}()

		if verifier, ok := mCopy.(acct.Verifier); ok {
			if err = verifier.Verify(r.Header.Get(requestHeader), trans); err != nil {
				return FullHttpErrorAndDebug(err, m.TableName(), "verifier validation")
			}
		}

		if merge, ok := mCopy.(MergeModel); ok {
			if err = merge.Merge(from); err != nil {
				return httperr.New(http.StatusBadRequest, err.Error(), err)
			}
		}

		if _, err = trans.Update(mCopy); err != nil {
			return clientError(err)
		}
		if e, ok := mCopy.(basemodel.Expander); ok {
			if err = e.Expand(db, ""); err != nil {
				return err
			}
		}
		return json.NewEncoder(w).Encode(mCopy)
	}
}

func DeleteByID(m MergeModel) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error {
		user := acct.GetCurrentRequestUserFromCache(r.Header.Get(requestHeader))
		requiredPermission := user.Group.GetPermission(m.TableName(), Write)
		if requiredPermission == acct.NoPerm {
			err := errors.New("mware: inadequate permissions for request")
			return httperr.New(http.StatusForbidden, "Inadequate permissions for request.", err)
		}
		params := mux.Vars(r)
		mCopy := copyResource(m)
		if err := GetID(db, user, mCopy, params["id"], requiredPermission); err != nil {
			return err
		}
		trans, err := db.Begin()
		defer func() {
			if err != nil {
				if err = trans.Rollback(); err != nil {
					logger.ErrorMsgf("unable to rollback %s create transaction: %s", m.TableName(), err.Error())
				}
			} else {
				if err = trans.Commit(); err != nil {
					logger.ErrorMsgf("unable to commit %s create transaction: %s", m.TableName(), err.Error())
				}
			}
		}()

		if verifier, ok := mCopy.(acct.Verifier); ok {
			if err = verifier.Verify(r.Header.Get(requestHeader), trans); err != nil {
				return FullHttpErrorAndDebug(err, m.TableName(), "verifier validation")
			}
		}

		if err = mCopy.Delete(trans); err != nil {
			return err
		}

		if _, err = trans.Update(mCopy); err != nil {
			return clientError(err)
		}
		if e, ok := mCopy.(basemodel.Expander); ok {
			if err = e.Expand(db, ""); err != nil {
				return err
			}
		}
		return json.NewEncoder(w).Encode(mCopy)
	}
}

func copyResource(m basemodel.Model) basemodel.Model {
	ptr := reflect.New(reflect.TypeOf(m).Elem())
	iFace := ptr.Interface().(basemodel.Model)
	return iFace
}

func FullHttpErrorAndDebug(err error, model, location string) error {
	message := fmt.Sprintf("%s did not pass %s", model, location)
	logger.Debugf("Error message: %s\nError: %s", message, err)
	return httperr.New(http.StatusUnauthorized, message, err)
}

func GetID(dbMap *gorp.DbMap, user *acct.User, m basemodel.Model, id interface{}, requiredPermission acct.Permission) error {
	builder := squirrel.Select("*").
		From(m.TableName()).
		Where(squirrel.Eq{m.TableName() + ".ID": id})
	builder = permissions.UserLevelFilter(builder, m, user, requiredPermission)
	query, args, _ := builder.ToSql()
	if err := dbMap.SelectOne(m, query, args...); err != nil {
		message := fmt.Sprintf("Could not find %s.", m.TableName())
		return httperr.New(http.StatusNotFound, message, err)
	}
	return nil
}

func clientError(err error) error {
	message := "Problem performing request.  Please alert the Account owner if the problem continues."
	return httperr.New(http.StatusBadRequest, message, err)
}
