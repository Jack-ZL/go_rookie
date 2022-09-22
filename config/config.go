/**
 * Package config
 * @Description: 支持toml格式配置文件
 */
package config

import (
	"flag"
	"github.com/BurntSushi/toml"
	grLog "github.com/Jack-ZL/go_rookie/log"
	"os"
)

var Conf = &GrConfig{
	logger: grLog.Default(),
}

type GrConfig struct {
	logger   *grLog.Logger
	Log      map[string]any
	Pool     map[string]any
	Template map[string]any
}

/**
 * init
 * @Author：Jack-Z
 * @Description: 初始化
 */
func init() {
	loadToml()
}

/**
 * loadToml
 * @Author：Jack-Z
 * @Description: 加载配置文件
 */
func loadToml() {
	confFile := flag.String("conf", "conf/app.toml", "app config file")
	flag.Parse()
	if _, err := os.Stat(*confFile); err != nil {
		Conf.logger.Info("conf/app.toml file not load，because not exist")
		return
	}

	_, err := toml.DecodeFile(*confFile, Conf)
	if err != nil {
		Conf.logger.Info("conf/app.toml decode fail check format")
		return
	}
}
