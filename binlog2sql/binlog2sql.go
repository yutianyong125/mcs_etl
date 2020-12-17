package binlog2sql

import (
	"fmt"
	mysql "github.com/siddontang/go-mysql/replication"
	"github.com/yutianyong125/mcs_etl/db"
	"github.com/yutianyong125/mcs_etl/util"
	"strings"
)

type Binlog2sql struct {
	columnSchemas map[string][]string
}

func NewBinlog2sql() *Binlog2sql {
	b := new(Binlog2sql)
	b.columnSchemas = make(map[string][]string)
	return b
}

// 解析binlog event
func (b *Binlog2sql) ParseEvent(eventType mysql.EventType, event mysql.Event) string {
	var sql string
	switch  eventType {
	case mysql.ROTATE_EVENT:
	case mysql.FORMAT_DESCRIPTION_EVENT:
		break
	case mysql.WRITE_ROWS_EVENTv2:
		if rowsEvent, ok := event.(*mysql.RowsEvent); ok {
			tableMap := rowsEvent.Table
			schema := string(tableMap.Schema)
			table := string(tableMap.Table)
			b.getColumnSchemas(schema, table)
			columns := b.columnSchemas[table]
			columnStr := ""
			valueStr := ""
			for _, column := range columns{
				columnStr += "`" + column + "`" + ","
			}
			for _, rows := range rowsEvent.Rows {
				for _, value := range rows {
					valueStr += processValue(value) + ","
				}
			}
			sql = fmt.Sprintf("insert into `%s`.`%s` (%s) values (%s)", schema, table,
				strings.Trim(columnStr, ","),
				strings.Trim(valueStr, ","),
			)
		}
		break
	case mysql.UPDATE_ROWS_EVENTv2:
		if rowsEvent, ok := event.(*mysql.RowsEvent); ok {
			tableMap := rowsEvent.Table
			schema := string(tableMap.Schema)
			table := string(tableMap.Table)
			b.getColumnSchemas(schema, table)
			columns := b.columnSchemas[table]
			var beforeUpdate, afterUpdate []string
			for i, value := range rowsEvent.Rows[0]{
				beforeUpdate = append(beforeUpdate, fmt.Sprintf("`%s` = %s", columns[i], processValue(value)))
			}
			for i, value := range rowsEvent.Rows[1]{
				afterUpdate = append(afterUpdate, fmt.Sprintf("`%s` = %s", columns[i], processValue(value)))
			}
			sql = fmt.Sprintf("update `%s`.`%s` set %s where %s", schema, table,
				strings.Join(afterUpdate, ","),
				strings.Join(beforeUpdate, " and "),
			)
		}
		break
	case mysql.DELETE_ROWS_EVENTv2:
		if rowsEvent, ok := event.(*mysql.RowsEvent); ok {
			tableMap := rowsEvent.Table
			schema := string(tableMap.Schema)
			table := string(tableMap.Table)
			b.getColumnSchemas(schema, table)
			columns := b.columnSchemas[table]
			var beforeDelete []string
			for _, rows := range rowsEvent.Rows {
				for i, value := range rows {
					beforeDelete = append(beforeDelete, fmt.Sprintf("`%s` = %s", columns[i], processValue(value)))
				}
			}
			sql = fmt.Sprintf("delete from `%s`.`%s` where %s", schema, table,
				strings.Join(beforeDelete, " and "),
			)
		}
		break
	case mysql.QUERY_EVENT: // ddl语句
		if queryEvent, ok := event.(*mysql.QueryEvent); ok {
			sql = string(queryEvent.Query)
			if sql == "BEGIN" {
				sql = ""
			} else {
				sql = "use " + string(queryEvent.Schema) + ";\n" + sql
			}
		}
		break
	}
	return sql
}

// 获取表的列信息
func (b *Binlog2sql) getColumnSchemas(schema string, table string) {
	if _, ok := b.columnSchemas[table]; ok {
		return
	}
	sql := fmt.Sprintf(`SELECT
                        COLUMN_NAME
                    FROM
                        information_schema.columns
                    WHERE
                        table_schema = '%s' AND table_name = '%s'`,
                        schema, table,
	)
	rows, err := db.SourceConn.Query(sql)
	util.CheckErr(err)
	var column string
	for rows.Next() {
		err = rows.Scan(&column)
		util.CheckErr(err)
		b.columnSchemas[table] = append(b.columnSchemas[table], column)
	}
}

func processValue (value interface{}) string{
	if value == nil {
		return "null"
	}
	return fmt.Sprintf("'%v'", value)
}
