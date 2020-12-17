package etl

import (
	"context"
	"fmt"
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
	"github.com/yutianyong125/mcs_etl/binlog2sql"
	"github.com/yutianyong125/mcs_etl/db"
	"github.com/yutianyong125/mcs_etl/env"
	"github.com/yutianyong125/mcs_etl/util"
)

type IncrementEtl struct {

}

func NewIncrementEtl () *IncrementEtl {
	return new(IncrementEtl)
}

func (incr *IncrementEtl) Run() {
	config := env.Config()
	cfg := replication.BinlogSyncerConfig {
		ServerID: config.IncrementEtl.ServerId,
		Flavor:   "mysql",
		Host:     config.Source.Host,
		Port:     config.Source.Port,
		User:     config.Source.User,
		Password: config.Source.Pwd,
	}
	syncer := replication.NewBinlogSyncer(cfg)

	streamer, _ := syncer.StartSync(mysql.Position{
		Name: config.IncrementEtl.StartFile,
		Pos: config.IncrementEtl.StartPosition,
	})

	for {
		ev, _ := streamer.GetEvent(context.Background())
		//ev.Dump(os.Stdout)
		b := binlog2sql.NewBinlog2sql()
		sql := b.ParseEvent(ev.Header.EventType, ev.Event)
		// 偏移量
		config.IncrementEtl.StartPosition = ev.Header.LogPos
		// 日志轮替处理
		if ev.Header.EventType == replication.ROTATE_EVENT {
			if rotateEvent, ok :=  ev.Event.(*replication.RotateEvent); ok {
				fmt.Println(string(rotateEvent.NextLogName), uint32(rotateEvent.Position))
				config.IncrementEtl.StartFile = string(rotateEvent.NextLogName)
				config.IncrementEtl.StartPosition = uint32(rotateEvent.Position)
			}
		}
		if sql == "" {
			continue
		}
		fmt.Println("binlog恢复的语句：")
		fmt.Println(sql)
		fmt.Println("==>")
		sql = TransformSql(sql)
		fmt.Println("兼容处理转换后的语句：")
		fmt.Println(sql)
		fmt.Println()
		fmt.Println("执行结果：")
		_, err := db.TargetConn.Exec(sql)
		util.CheckErr(err)
		// 保存进度，暴力保存到原配置文件中
		env.Save(config)
		fmt.Println("执行成功")
		fmt.Println()
	}
}
