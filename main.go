package main

import (
	"encoding/base64"
	"flag"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"git.fiblab.net/sim/protos/v2/go/city/economy/v2/economyv2connect"
	"git.fiblab.net/sim/simulet-go/ecosim"
	"git.fiblab.net/sim/simulet-go/task"
	"git.fiblab.net/sim/simulet-go/utils/config"
	"git.fiblab.net/sim/syncer/v3"
	easy "git.fiblab.net/utils/logrus-easy-formatter"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	// 分布式模式syncer地址，如果设置为空则激活独立部署模式
	// 独立部署：不需要syncer，不向其他服务提供受保护的RPC访问
	syncerAddr = flag.String("syncer", "", "syncer address (empty means standalone mode), e.g. http://localhost:53001")
	// 模拟任务名，主要用于etcd中服务注册与输出的数据库表名前缀
	job = flag.String("job", "job0", "the name of the whole simulation task")
	// 本程序监听的gRPC地址
	grpcAddr = flag.String("listen", ":51102", "gRPC listening address")
	// 配置文件路径
	configPath = flag.String("config", "", "config file path")
	// 配置文件Base64编码后的数据
	configData = flag.String("config-data", "", "config file base64 encoded data")
	// 数据加载input的缓存地址，设置为空则禁用缓存功能
	// 缓存：将proto数据根据数据库db和col序列化到本地文件系统，并总是先试图从文件系统中加载
	cacheDir = flag.String("cache", "data/", "input cache dir path (empty means disable cache)")

	// log
	logLevels = map[string]logrus.Level{
		"trace":    logrus.TraceLevel,
		"debug":    logrus.DebugLevel,
		"info":     logrus.InfoLevel,
		"warn":     logrus.WarnLevel,
		"error":    logrus.ErrorLevel,
		"critical": logrus.FatalLevel,
		"off":      logrus.PanicLevel,
	}
	logLevel = flag.String("log.level", "info", "日志级别（可选项：trace debug info warn error critical off）")

	log       = logrus.WithField("module", "simulet")
	syncerLog = logrus.WithField("module", "syncer")
)

func main() {
	flag.Parse()
	logrus.SetFormatter(&easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05.0000",
		LogFormat:       "[%module%] [%time%] [%lvl%] %msg%\n",
	})
	// log: 运行时才修改
	if level, ok := logLevels[*logLevel]; ok {
		logrus.SetLevel(level)
	} else {
		log.Panicf("log.level must be one of %v", logLevels)
	}
	// 获取配置
	var c config.Config
	var file []byte
	var err error
	if *configPath != "" {
		file, err = os.ReadFile(*configPath)
		if err != nil {
			log.Panicf("config file load err: %v", err)
		}
	} else if *configData != "" {
		file, err = base64.StdEncoding.DecodeString(*configData)
		if err != nil {
			log.Panicf("config data load err: %v", err)
		}
	} else {
		log.Panic("config file or config data must be specified")
	}
	if err := yaml.UnmarshalStrict(file, &c); err != nil {
		log.Panicf("config file load err: %v", err)
	}
	log.Infof("%+v", c)

	sidecar := syncer.NewSidecar(task.SelfName, *grpcAddr, *syncerAddr)
	t := task.NewContext(
		*job,
		*grpcAddr,
		*syncerAddr,
		syncerLog,
		*cacheDir,
		c,
		sidecar,
		true,
	)

	// 创建经济模拟器实例
	economySimulator := ecosim.NewServer()

	// 注册经济模拟器服务
	sidecar.Register(
		economyv2connect.OrgServiceName,
		func(opts ...connect.HandlerOption) (pattern string, handler http.Handler) {
			return economyv2connect.NewOrgServiceHandler(economySimulator, opts...)
		},
		syncer.WithNoLock(),
	)

	t.Run()
}
