package rest

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

type operator string

const (
	lt      operator = "lt"
	lte     operator = "lte"
	gt      operator = "gt"
	gte     operator = "gte"
	in      operator = "in"
	like    operator = "like"
	nin     operator = "nin"
	between operator = "bet"
)

type order string

const (
	asc  order = "asc"
	desc order = "desc"
)

const (
	KeyOrder   = "q-order"
	KeyLimit   = "q-limit"
	KeyOffset  = "q-offset"
	KeyExpand  = "q-expand"
	KeyExclude = "q-exclude"
)

const (
	LimitDefault  = 10000
	OrderDefault  = desc
	ColumnDefault = "id"
)

type quantifiedQuery struct {
	CountQuery squirrel.SelectBuilder
	Query      squirrel.SelectBuilder
}

func Query(src interface{}, tableName string, values url.Values) (*quantifiedQuery, error) {
	q := &quantifiedQuery{
		CountQuery: squirrel.Select("COUNT(base.*)").From(fmt.Sprintf("%s as base", tableName)),
		Query:      squirrel.Select("base.*").From(fmt.Sprintf("%s as base", tableName)),
	}

	// Check for deleted flag, if not present default to deleted = false.
	if values.Get("in-deleted") == "" && values.Get("deleted") == "" {
		q.CountQuery = q.CountQuery.Where(squirrel.Eq{"base.Deleted": false})
		q.Query = q.Query.Where(squirrel.Eq{"base.Deleted": false})
	}

	if values.Get(KeyOffset) != "" {
		offset, err := strconv.ParseUint(values.Get(KeyOffset), 10, 64)
		if err != nil {
			return q, errors.Wrap(err, "error in parsing offset from string values")
		}
		q.Query = q.Query.Offset(offset)
	}

	// Set the WHERE clause.
	var err error
	for key, value := range values {
		q.CountQuery, err = WhereValueForKey(q.CountQuery, src, key, value[0])
		if err != nil {
			return nil, err
		}
		q.Query, err = WhereValueForKey(q.Query, src, key, value[0])
		if err != nil {
			return nil, err
		}
	}

	// Set the ORDER BY clause.
	q.CountQuery, err = setOrderBy(src, values, q.CountQuery, "base.")
	if err != nil {
		return nil, err
	}
	q.Query, err = setOrderBy(src, values, q.Query, "base.")
	if err != nil {
		return nil, err
	}

	return q, nil
}

func setOrderBy(src interface{}, values url.Values, builder squirrel.SelectBuilder, alias ...interface{}) (squirrel.SelectBuilder, error) {
	aliasString := ""
	if len(alias) > 0 {
		aliasString = alias[0].(string)
	}
	if oVal := values.Get(KeyOrder); oVal != "" {
		field, order, err := orderFromValue(src, oVal)
		if err != nil {
			return builder, err
		}
		builder = builder.OrderBy(aliasString + field + " " + string(order))
	} else {
		builder = builder.OrderBy(aliasString + ColumnDefault + " " + string(OrderDefault))
	}
	return builder, nil
}

func WhereValueForKey(builder squirrel.SelectBuilder, iFace interface{}, key, value string, alias ...interface{}) (squirrel.SelectBuilder, error) {
	aliasString := ""
	if len(alias) > 0 {
		aliasString = alias[0].(string)
	}
	keyParts := strings.Split(key, "-")
	sqlString := ""

	switch len(keyParts) {
	case 1:
		if !structHasField(iFace, keyParts[0]) {
			return builder, nil
		}
		sqlString = aliasString + keyParts[0] + " = ?"
	case 2:
		op := operator(keyParts[0])
		field := keyParts[1]
		if !structHasField(iFace, field) {
			return builder, nil
		}
		field = aliasString + field
		switch op {
		case lt:
			sqlString = field + " < ?"
		case lte:
			sqlString = field + " <= ?"
		case gt:
			sqlString = field + " > ?"
		case gte:
			sqlString = field + " >= ?"
		case like:
			sqlString = field + " LIKE ?"
		case in:
			values := strings.Split(value, ",")
			builder = builder.Where(squirrel.Eq{"base." + field: values})
			return builder, nil
		case nin:
			values := strings.Split(value, ",")
			for _, v := range values {
				builder = builder.Where("NOT "+"base."+field+" <=> ?", v)
			}
			return builder, nil
		default:
			return builder, nil
		}
	case 3:
		op := operator(keyParts[0])
		fieldStart := keyParts[1]
		fieldEnd := keyParts[2]
		if !structHasField(iFace, fieldStart) {
			return builder, nil
		}
		if !structHasField(iFace, fieldEnd) {
			return builder, nil
		}
		fieldStart = aliasString + fieldStart
		fieldEnd = aliasString + fieldEnd
		switch op {
		case between:
			values := strings.Split(value, "-")
			whereCluase := fmt.Sprintf("( (%s >= %s AND %s <= %s) OR (%s >= %s AND %s <= %s) OR ( %s <= %s AND %s > 0 AND (%s = 0 OR %s >= %s)))",
				"base."+fieldStart, values[0], "base."+fieldStart, values[1],
				"base."+fieldEnd, values[0], "base."+fieldEnd, values[1],
				fieldStart, values[0], fieldStart, fieldEnd, fieldEnd, values[1])
			builder = builder.Where(whereCluase)
			return builder, nil
		default:
			return builder, nil
		}
	default:
		return builder, nil
	}

	builder = builder.Where(sqlString, value)
	return builder, nil
}

func orderFromValue(src interface{}, value string) (string, order, error) {
	parts := strings.Split(value, "-")

	// return error if not in the format asc-field
	if len(parts) != 2 {
		return "", asc, errors.New("mware: q-order format invalid")
	}

	// return error field isn't in struct
	if !structHasField(src, parts[1]) {
		message := fmt.Sprintf("mware: q-order field, %v, isn't present in model", parts[1])
		return "", asc, errors.New(message)
	}

	// return error if ordering is invalid
	order := order(parts[0])
	if order != asc && order != desc {
		message := fmt.Sprintf("mware: q-order ordering, %v, is invalid.  Must use asc or desc.", parts[0])
		return "", asc, errors.New(message)
	}

	return parts[1], order, nil
}

func structHasField(src interface{}, key string) bool {
	objT := reflect.Indirect(reflect.ValueOf(src)).Type()
	for i := 0; i < objT.NumField(); i++ {
		field := objT.Field(i)
		if isStruct(field) {
			if structHasField(reflect.Zero(field.Type).Interface(), key) {
				return true
			}
		} else {
			name := strings.ToLower(field.Name)
			if name == strings.ToLower(key) {
				return true
			}
		}
	}
	return false
}

func isStruct(field reflect.StructField) bool {
	return reflect.Indirect(reflect.Zero(field.Type)).Kind() == reflect.Struct
}

func UintFromKey(values url.Values, key string, d uint64) uint64 {
	v := values.Get(key)
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil || i < 0 {
		return d
	}
	return uint64(i)
}
