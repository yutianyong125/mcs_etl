package main

import (
	"flag"
	"fmt"
	"github.com/yutianyong125/mcs_etl/etl"
)

func main() {
	model := flag.String("model", "", "etl模式，-model full: 全量etl, -model increment: 增量etl")
	flag.Parse()
	if *model == "" {
		fmt.Println("请指定model参数，-h 选项查看说明")
		return
	}
	switch *model {
	case "increment":
		// 增量ETL模式
		incrementEtl := etl.NewIncrementEtl()
		incrementEtl.Run()
		return
	case "full":
		fullEtl := etl.NewFullEtl()
		fullEtl.Run()
		return
	default:
		fmt.Println("请指定model参数，-h 选项查看说明")
		return
	}
}


