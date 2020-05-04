package acct

import (
	"github.com/rahul2393/small-assignment-server/cache"
)

const (
	reqUserKey = "req-user"
)

func GetCurrentRequestUserFromCache(reqID string) *User {
	src, in := cache.Get(reqUserKey, reqID)
	if !in {
		return nil
	}
	return src.(*User)
}

func ReqSetUser(reqID string, user *User) {
	cache.Set(reqUserKey, reqID, cache.Item{Src: user})
}

func ReqDeleteUser(reqID string) {
	cache.Delete(reqUserKey, reqID)
}
