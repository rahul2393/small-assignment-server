package permissions

import (
	"github.com/Masterminds/squirrel"
	"github.com/rahul2393/small-assignment-server/models/acct"
	"github.com/rahul2393/small-assignment-server/models/basemodel"
)

func UserLevelFilter(builder squirrel.SelectBuilder, m basemodel.Model, user *acct.User, requiredPermission acct.Permission) squirrel.SelectBuilder {
	if !requiredPermission.IsAll {
		key := ""
		if m.TableName() == acct.TableNameUser {
			key = "ID"
		} else {
			key = "UserID"
		}
		builder = builder.Where(squirrel.Eq{key: user.ID})
	}
	return builder
}
