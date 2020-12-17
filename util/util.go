package util

import (
	"fmt"
	"os"
	"time"
)

// 工具函数包

func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}

func Elapsed(work string) func() {
	start := time.Now()
	return func () {
		fmt.Printf("%s 耗时 %v\n", work, time.Since(start))
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}