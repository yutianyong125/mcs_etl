package etl

import (
	"regexp"
	"strings"
)

// schema转换处理
func TransformSql(content string) string{

	content = strings.Replace(content, "InnoDB", "Columnstore", -1)
	content = strings.Replace(content, "MyISAM", "Columnstore", -1)
	content = strings.Replace(content, "NOT NULL", "", -1)
	content = strings.Replace(content, "timestamp", "datetime", -1)
	content = strings.Replace(content, "json", "text", -1)
	content = strings.Replace(content, "ROW_FORMAT=COMPACT", "", -1)
	content = strings.Replace(content, "ROW_FORMAT=DYNAMIC", "", -1)
	content = strings.Replace(content, "DEFAULT CURRENT_TIMESTAMP", "", -1)
	content = strings.Replace(content, "COLLATE utf8mb4_bin", "", -1)
	content = strings.Replace(content, "ON UPDATE CURRENT_TIMESTAMP", "", -1)

	reg := regexp.MustCompile(`.*KEY.*`)
	content = reg.ReplaceAllString(content, "")

	reg = regexp.MustCompile(`,\s*\)`)
	content = reg.ReplaceAllString(content, "\n)")

	reg = regexp.MustCompile(`AUTO_INCREMENT(=[0-9]*)?`)
	content = reg.ReplaceAllString(content, "")

	reg = regexp.MustCompile(`<COLLATE.*utf8_bin>`)
	content = reg.ReplaceAllString(content, "")

	reg = regexp.MustCompile(`<timestamp>.*TIMESTAMP|<timestamp>`)
	content = reg.ReplaceAllString(content, "")

	reg = regexp.MustCompile(`<CHARACTER.*SET.*[utf8|utf8mb4]>`)
	content = reg.ReplaceAllString(content, "")

	reg = regexp.MustCompile(`decimal\([0-9]*`)
	content = reg.ReplaceAllString(content, "decimal(18")

	reg = regexp.MustCompile(`bit\(([0-9])\)`)
	content = reg.ReplaceAllString(content, "int(${1})")

	reg = regexp.MustCompile(`b(\'[0-9]\')`)
	content = reg.ReplaceAllString(content, "${1}")

	reg = regexp.MustCompile(`COLLATE.* utf8mb4_unicode_ci`)
	content = reg.ReplaceAllString(content, "")

	reg = regexp.MustCompile(`/\*.*\*/;`)
	content = reg.ReplaceAllString(content, "")

	return content
}
