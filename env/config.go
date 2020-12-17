package env

import (
	"github.com/BurntSushi/toml"
	"github.com/yutianyong125/mcs_etl/util"
	"os"
	"sync"
)

type tomlConfig struct {
	IncrementEtl IncrementEtl
	FullEtl FullEtl
	Rules []*Rule `toml:"rule"`
	Source Source
	Target Target
}

type IncrementEtl struct {
	StartFile string
	StartPosition uint32
	ServerId uint32
}

type FullEtl struct {
	MysqlBinDir string
	OutFileDir string
}

type Rule struct {
	Schema string
	Tables []string
}

type Source struct {
	Host string
	Port uint16
	User string
	Pwd string
}

type Target struct {
	Host string
	Port uint16
	User string
	Pwd string
}

var (
	config *tomlConfig
	once sync.Once
	configFile = "conf/etl.toml"
)

func init() {
	if ok, _ := util.PathExists(configFile); !ok {
		panic("配置文件conf/etl.toml不存在")
	}
}

func Config() *tomlConfig {
	once.Do(func() {
		_, err := toml.DecodeFile(configFile, &config)
		util.CheckErr(err)
	})
	return config
}

func Save(config *tomlConfig) {
	f, err := os.OpenFile(configFile, os.O_RDWR,0644)
	util.CheckErr(err)
	err = toml.NewEncoder(f).Encode(config)
	util.CheckErr(err)
}
