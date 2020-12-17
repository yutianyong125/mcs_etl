package etl

import (
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/yutianyong125/mcs_etl/db"
	"github.com/yutianyong125/mcs_etl/env"
	"github.com/yutianyong125/mcs_etl/util"
	"os"
	"sync"
)

type FullEtl struct {

}

var wg = sync.WaitGroup{}
var jobsChan = make(chan struct{}, 10)

func NewFullEtl() *FullEtl{
	return new(FullEtl)
}

func (f *FullEtl) Run() {
	config := env.Config()
	// 创建outFileDir
	err := os.MkdirAll(config.FullEtl.OutFileDir, os.ModePerm)
	util.CheckErr(err)
	defer util.Elapsed("fullEtl")()
	var table string
	tables := make([][]string, 0)

	for _, rule := range config.Rules {
		// 创建目标数据库
		// 先删除已有数据库
		_, err := db.TargetConn.Exec("drop database if exists " + rule.Schema)
		util.CheckErr(err)
		// 创建数据库
		_, err = db.TargetConn.Exec("create database " + rule.Schema)
		util.CheckErr(err)

		if len(rule.Tables) == 1 && rule.Tables[0] == "*" {
			// 查询表
			result, err := db.SourceConn.Query( "use " + rule.Schema + ";show tables")
			util.CheckErr(err)
			for result.Next() {
				result.Scan(&table)
				tables = append(tables, []string{rule.Schema, table})
				//jobsChan <- struct{}{}
				//wg.Add(1)
				//go doSync(rule.Schema, table)
			}
			result.Close()
		} else {
			for _, table := range rule.Tables {
				tables = append(tables, []string{rule.Schema, table})
				//jobsChan <- struct{}{}
				//wg.Add(1)
				//go doSync(rule.Schema, table)
			}
		}
	}

	for _, tableInfo := range tables {
		jobsChan <- struct{}{}
		wg.Add(1)
		go doSync(tableInfo[0], tableInfo[1])
	}

	wg.Wait()
}

func doSync(database string, table string) {
	defer util.Elapsed(fmt.Sprintf("同步`%s`.`%s`", database, table))()
	defer wg.Done()
	// 导出表结构
	createSql := dumpSchema(database, table)
	// 转换sql
	createSql = TransformSql(createSql)
	// 切换数据库
	_, err := db.TargetConn.Exec("use " + database)
	util.CheckErr(err)
	// 创建目标数据库中的表
	_, err = db.TargetConn.Exec(createSql)
	util.CheckErr(err)
	// 导出源库数据
	outFileEtl(database, table)
	// 目标库导入数据
	loadFileEtl(database, table)

	<-jobsChan
}

// 导出数据表结构
func dumpSchema(database string, table string) string {
	sql := fmt.Sprintf("show create table `%s`.`%s`", database, table)
	result, err := db.SourceConn.Query(sql)
	util.CheckErr(err)
	var tmp, createSql string
	for result.Next() {
		_ = result.Scan(&tmp, &createSql)
	}
	result.Close()
	return createSql
}

// 导出源库数据
func outFileEtl(database string, table string) {
	//defer util.Elapsed("outFileEtl")()
	config := env.Config()
	outFileDir := config.FullEtl.OutFileDir
	fileName := table + ".csv"
	filePath := outFileDir + fileName
	if ok, _ := util.PathExists(filePath); !ok {
		outFileSql := fmt.Sprintf("select * from `%s`.`%s` INTO OUTFILE '%s' CHARACTER SET utf8mb4 FIELDS TERMINATED BY '&' ENCLOSED BY '\"';",
			database,
			table,
			filePath,
		)
		_, err := db.SourceConn.Exec(outFileSql)
		util.CheckErr(err)
	}
}

// 目标数据库导入数据
func loadFileEtl(database string, table string) {
	//defer util.Elapsed("loadFileEtl")()
	config := env.Config()
	outFileDir := config.FullEtl.OutFileDir
	fileName := table + ".csv"
	filePath := outFileDir + fileName
	if ok, _ := util.PathExists(filePath); ok {
		mysql.RegisterLocalFile(filePath)
		inFileSql := fmt.Sprintf("LOAD DATA LOCAL INFILE '%s' INTO TABLE `%s`.`%s` CHARACTER SET utf8mb4 FIELDS TERMINATED BY '&' ENCLOSED BY '\"';",
			filePath,
			database,
			table,
		)
		_, err := db.TargetConn.Exec(inFileSql)
		util.CheckErr(err)
	}
}