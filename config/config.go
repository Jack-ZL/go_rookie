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
	logger    *grLog.Logger
	Log       map[string]any //日志保存路径配置
	Pool      map[string]any //线程池数量配置
	Template  map[string]any //模板文件配置
	Mysql     map[string]any //mysql数据库配置
	Sqlserver map[string]any //sqlserver数据库配置
	Redis     map[string]any //redis配置
	Es        map[string]any //es配置
	Mongodb   map[string]any //Mongodb配置
	App       map[string]any //应用基本配置
	Url       map[string]any //链接类配置
	Thirdconf map[string]any //其他第三方的配置
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
