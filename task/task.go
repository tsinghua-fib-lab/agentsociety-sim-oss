package task

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"git.fiblab.net/sim/syncer/v3"
	"github.com/sirupsen/logrus"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/clock"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity/aoi"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity/junction"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity/lane"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity/person"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity/person/route"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/entity/road"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/utils/config"
	"github.com/tsinghua-fib-lab/agentsociety-sim-oss/utils/input"
)

// waitForServerReady 等待服务器就绪
// 功能：通过HTTP请求检查服务器是否已经启动并可以响应
// 参数：addr-服务器地址，retryCount-重试次数，interval-重试间隔
// 返回：错误信息，如果服务器就绪则返回nil
// 算法说明：
// 1. 创建HTTP客户端，设置超时时间
// 2. 循环发送GET请求到指定地址
// 3. 如果请求成功，关闭响应体并返回nil
// 4. 如果请求失败，等待指定间隔后重试
// 5. 达到最大重试次数后返回错误
func waitForServerReady(addr string, retryCount int, interval time.Duration) error {
	client := &http.Client{
		Timeout: interval,
	}
	for range retryCount {
		resp, err := client.Get(addr)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("server `%v` did not become ready after %d retries", addr, retryCount)
}

// Context 仿真任务上下文
// 功能：包含一次仿真任务的所有变量和状态，替代原来的全局变量
// 说明：管理仿真系统的所有组件，包括时钟、管理器、配置、输出等
type Context struct {

	// 任务名
	job string
	// 关闭指令
	closed atomic.Bool

	// 时钟
	clock *clock.Clock

	// 辅助程序，处理分布式模式下相关调用，包括与syncer、其他服务的交互
	sidecar *syncer.Sidecar
	// 内部创建的sidecar 返回供其他服务注册
	WithinSidecar *syncer.Sidecar
	// sidecar close channel
	sidecarCloseCh chan struct{}
	// 缓存文件夹
	cacheDir string

	// Lane管理器
	laneManager entity.ILaneManager
	// Aoi管理器
	aoiManager entity.IAoiManager
	// Road管理器
	roadManager entity.IRoadManager
	// Junction管理器
	junctionManager entity.IJunctionManager
	// Person管理器
	personManager entity.IPersonManager

	// 运行时配置文件
	runtimeConfig *config.RuntimeConfig
	// 导航服务
	router entity.IRouter

	// 用于初始化的输入
	initRes *input.Input
}

// NewContext 创建新的仿真任务上下文
// 功能：初始化仿真系统的所有组件和配置
// 参数：
//   - job: 任务名称
//   - grpcAddr: gRPC服务地址
//   - syncerAddr: syncer服务地址
//   - syncerLog: syncer日志记录器
//   - cacheDir: 缓存目录
//   - c: 配置对象
//   - sidecar: 外部sidecar实例
//   - startSidecarServe: 是否启动sidecar服务
//
// 返回：初始化完成的Context实例
// 算法说明：
// 1. 根据配置决定是否启动内部syncer
// 2. 创建Context实例并设置基本属性
// 3. 初始化时钟、统计、输出等功能
// 4. 下载和初始化地图数据
// 5. 创建各种管理器（车道、POI、AOI、道路、路口、人员、线路、出租车）
// 6. 注册RPC服务到sidecar
// 7. 启动sidecar服务（如果需要）
func NewContext(
	job string,
	grpcAddr string,
	syncerAddr string,
	syncerLog *logrus.Entry,
	cacheDir string,
	c config.Config,
	sidecar *syncer.Sidecar,
	startSidecarServe bool,
) *Context {
	// 启动内部syncer
	var WithinSidecar *syncer.Sidecar
	ctx := &Context{
		job:      job,
		cacheDir: cacheDir,
		// sidecar:        ,
		sidecar:        sidecar,
		WithinSidecar:  WithinSidecar,
		sidecarCloseCh: make(chan struct{}),
	}
	ctx.clock = clock.New(c.Control.Step)

	// 下载所有模拟器启动所需的数据
	ctx.initRes = input.Init(c, ctx.cacheDir)

	ctx.runtimeConfig = config.NewRuntimeConfig(c)

	// 新建各类模拟对象
	ctx.laneManager = lane.NewManager(ctx)
	ctx.aoiManager = aoi.NewManager(ctx)
	ctx.roadManager = road.NewManager(ctx)
	ctx.junctionManager = junction.NewManager(ctx)
	ctx.personManager = person.NewManager(ctx)

	ctx.clock.Register(ctx.sidecar)
	ctx.junctionManager.Register(ctx.sidecar)
	ctx.personManager.Register(ctx.sidecar)

	// sidecar协程，用于提供gRPC服务
	if startSidecarServe {
		go func() {
			err := ctx.sidecar.Serve()
			if err != nil {
				log.Panicf("failed to serve: %v", err)
			}
			ctx.sidecarCloseCh <- struct{}{}
		}()
	}

	return ctx
}

func (ctx *Context) GetInput() *input.Input {
	return ctx.initRes
}

func (ctx *Context) Clock() *clock.Clock {
	return ctx.clock
}

func (ctx *Context) LaneManager() entity.ILaneManager {
	return ctx.laneManager
}

func (ctx *Context) AoiManager() entity.IAoiManager {
	return ctx.aoiManager
}

func (ctx *Context) RoadManager() entity.IRoadManager {
	return ctx.roadManager
}

func (ctx *Context) JunctionManager() entity.IJunctionManager {
	return ctx.junctionManager
}

func (ctx *Context) PersonManager() entity.IPersonManager {
	return ctx.personManager
}

func (ctx *Context) RuntimeConfig() *config.RuntimeConfig {
	return ctx.runtimeConfig
}

func (ctx *Context) Router() entity.IRouter {
	return ctx.router
}

func (ctx *Context) Init() {
	ctx.clock.Init()

	initRes := ctx.initRes
	// 数据加载
	mapData := initRes.Map
	persons := initRes.Persons.Persons

	log.Infof("Lane: %v", len(mapData.Lanes))
	log.Infof("Road: %v", len(mapData.Roads))
	log.Infof("Junction: %v", len(mapData.Junctions))
	log.Infof("AOI: %v", len(mapData.Aois))
	log.Infof("Person: %v", len(persons))

	ctx.laneManager.Init(mapData.Lanes) // 先完成lane的所有初始化
	// 在建立好poi、lanes的基础上
	// AOI初始化
	ctx.aoiManager.Init(mapData.Aois, ctx.laneManager)
	// road初始化
	ctx.roadManager.Init(mapData.Roads, ctx.laneManager)
	// junction初始化
	ctx.junctionManager.Init(mapData.Junctions, ctx.laneManager, ctx.roadManager)
	// road初始化其中的前驱后继路口
	ctx.roadManager.InitAfterJunction(ctx.junctionManager)

	// 完成地图构建后，开始构建person
	ctx.personManager.Init(
		persons,
		mapData.Header,
		ctx.aoiManager, ctx.laneManager,
	)
	// router
	ctx.router = route.New(initRes)
}

func (ctx *Context) Close() {
	if ctx.closed.Load() {
		return
	}
	ctx.sidecar.Close()
	// wait for graceful stop
	<-ctx.sidecarCloseCh
	ctx.closed.Store(true)
}
