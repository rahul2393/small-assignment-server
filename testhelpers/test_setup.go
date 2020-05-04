package testhelpers

import (
	"fmt"
	"net/http"
	"testing"

	"gopkg.in/BurntSushi/toml.v0"
	"gopkg.in/gorp.v1"

	"github.com/rahul2393/small-assignment-server/dbutil"
	"github.com/rahul2393/small-assignment-server/httperr"
)

type TestHandler struct {
	T       *testing.T
	Db      *gorp.DbMap
	Handler func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap) error
}

func (h *TestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.Handler(w, r, h.Db); err != nil {
		httperr.Write(w, err)
		return
	}
}

func TestDBurl() string {
	type configFile struct {
		TestDsn string `toml:"test_dsn"`
	}
	cfg := &configFile{}
	if _, err := toml.DecodeFile("../conf.toml", &cfg); err != nil {
		fmt.Printf("errors is %v\n", err)
	}
	return cfg.TestDsn
}

func SetupTestWithFixtures() *gorp.DbMap {
	db, _ := dbutil.DbForURL(TestDBurl())
	dbutil.CleanAndSeedTestDB(db)
	return db
}
