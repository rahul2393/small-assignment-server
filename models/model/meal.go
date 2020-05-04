package model

import (
	"time"

	"gopkg.in/gorp.v1"

	"github.com/Masterminds/squirrel"
	"github.com/ShaleApps/gator"
	"github.com/pkg/errors"
	"github.com/rahul2393/small-assignment-server/milli"
	"github.com/rahul2393/small-assignment-server/models/acct"
	"github.com/rahul2393/small-assignment-server/models/basemodel"
)

const (
	ModelNameMeal = "Meal"
	TableNameMeal = "meals"
)

type Meal struct {
	basemodel.BaseModel

	UserID      int64  `json:"userID"`
	Description string `json:"description"`
	MealTime    int64  `json:"mealTime"`
	MealDate    int64  `json:"mealDate"`
	Calories    int64  `json:"calories"`

	User *acct.User `db:"-" json:"user,omitempty"`
}

func (meal *Meal) Merge(src interface{}) error {
	from, ok := src.(*Meal)
	if !ok {
		return acct.ErrMergeWrongType
	}
	meal.Description = from.Description
	meal.Calories = from.Calories
	meal.MealTime = from.MealTime
	meal.MealDate = from.MealDate
	return nil
}

func (meal *Meal) TableName() string {
	return TableNameMeal
}

func (meal *Meal) Delete(s gorp.SqlExecutor) error {
	meal.Deleted = true
	return nil
}

func (meal *Meal) PreInsert(s gorp.SqlExecutor) error {
	meal.Created = milli.Timestamp(time.Now())
	meal.Updated = milli.Timestamp(time.Now())
	if meal.UserID <= 0 {
		return errors.New("handler: meal requires userID")
	}
	if err := gator.NewStruct(meal).Validate(); err != nil {
		return errors.Wrap(err, "error in validating meal")
	}
	return nil
}

func (meal *Meal) PreUpdate(s gorp.SqlExecutor) error {
	if meal.MealTime == 0 {
		meal.MealTime = meal.Created
	}
	meal.Updated = milli.Timestamp(time.Now())
	if err := gator.NewStruct(meal).Validate(); err != nil {
		return errors.Wrap(err, "error in validating user")
	}
	return nil
}

func (meal *Meal) Expand(s gorp.SqlExecutor, exclude string) error {
	meal.ModelName = ModelNameMeal
	user := &acct.User{}
	query, args, _ := squirrel.Select("*").From(acct.TableNameUser).Where(squirrel.Eq{"ID": meal.UserID}).ToSql()
	if err := s.SelectOne(user, query, args...); err != nil {
		return err
	}
	meal.User = user
	return nil
}

func (meal *Meal) Verify(reqId string, s gorp.SqlExecutor) error {
	currentUser := acct.GetCurrentRequestUserFromCache(reqId)
	if currentUser.GroupID != acct.Admin.ID && meal.UserID != currentUser.ID {
		return errors.Wrapf(errors.New("invalid meal"), "verification failed")
	}
	return nil
}
