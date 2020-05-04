package dbutil

import (
	"bufio"
	"database/sql"
	"fmt"
	glog "log"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"gopkg.in/BurntSushi/toml.v0"
	"gopkg.in/gorp.v1"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rahul2393/small-assignment-server/models/acct"
	"github.com/rahul2393/small-assignment-server/models/model"
)

const (
	maxConnections     = 11
	maxIdleConnections = 11
)

var (
	globalDB *gorp.DbMap
	once     sync.Once
)

// DB returns a *gorp.DbMap which maps gorp a database connection pool.
// The underlying *gorp.DbMap is a singleton that is lazily loaded upon
// calling DB.
func DB() (*gorp.DbMap, error) {
	var initErr error
	once.Do(func() {
		type configFile struct {
			Dsn string
		}
		cfg := &configFile{}
		if _, err := toml.DecodeFile("./conf.toml", &cfg); err != nil {
			initErr = err
		}
		dbMap, err := DbForURL(cfg.Dsn)
		initErr = err
		globalDB = dbMap
	})
	return globalDB, initErr
}

func DbForURL(url string) (*gorp.DbMap, error) {
	db, err := sql.Open("mysql", url)
	if err != nil {
		glog.Printf("unable to open connection pool: %#v\n", err)
		return nil, err
	}
	db.SetMaxOpenConns(maxConnections)
	db.SetMaxIdleConns(maxIdleConnections)

	// set up table mapping
	dbMap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}}
	dbMap.AddTableWithName(acct.User{}, acct.TableNameUser).SetKeys(true, "ID")
	dbMap.AddTableWithName(acct.Token{}, acct.TableNameToken).SetKeys(true, "ID")
	dbMap.AddTableWithName(model.Meal{}, acct.TableNameMeal).SetKeys(true, "ID")

	// ping the db
	if err := dbMap.Db.Ping(); err != nil {
		glog.Printf("the db could not be pinged %#v\n", err)
	}

	return dbMap, nil
}

func CleanAndSeedTestDB(db *gorp.DbMap) {
	_, filename, _, _ := runtime.Caller(0)
	testFixturePath := path.Join(path.Dir(filename), "../sql/test_data.sql")
	file, err := os.Open(testFixturePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := scanner.Text()
		if strings.TrimSpace(s) == "" {
			continue
		}
		if _, err = db.Exec(s); err != nil {
			panic(fmt.Errorf("failed to exec: %s query: %s", err, s))
		}
	}
}
