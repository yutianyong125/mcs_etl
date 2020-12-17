package db

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/yutianyong125/mcs_etl/env"
	"github.com/yutianyong125/mcs_etl/util"
	"strconv"
	"time"
)

var SourceConn *sql.DB
var TargetConn *sql.DB

func init() {
	var err error
	config := env.Config()
	dsn := fmt.Sprintf(`%s:%s@(%s:%s)/?charset=utf8&parseTime=True&loc=Local&multiStatements=true`,
		config.Source.User, config.Source.Pwd, config.Source.Host, strconv.Itoa(int(config.Source.Port)))
	SourceConn, err = sql.Open("mysql", dsn)
	util.CheckErr(err)
	SourceConn.SetMaxIdleConns(10)
	SourceConn.SetMaxOpenConns(0)
	SourceConn.SetConnMaxLifetime(600 * time.Second)

	dsn = fmt.Sprintf(`%s:%s@(%s:%s)/?charset=utf8&parseTime=True&loc=Local&multiStatements=true`,
		config.Target.User, config.Target.Pwd, config.Target.Host, strconv.Itoa(int(config.Target.Port)))
	TargetConn, err = sql.Open("mysql", dsn)
	util.CheckErr(err)
	TargetConn.SetMaxIdleConns(10)
	TargetConn.SetMaxOpenConns(0)
	TargetConn.SetConnMaxLifetime(600 * time.Second)
}


