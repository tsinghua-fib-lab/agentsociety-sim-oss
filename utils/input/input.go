package input

import (
	"context"
	"fmt"
	"sync"

	"git.fiblab.net/general/common/v2/cache"
	"git.fiblab.net/general/common/v2/mongoutil"
	"git.fiblab.net/general/common/v2/protoutil"
	geov2 "git.fiblab.net/sim/protos/v2/go/city/geo/v2"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	personv2 "git.fiblab.net/sim/protos/v2/go/city/person/v2"
	tripv2 "git.fiblab.net/sim/protos/v2/go/city/trip/v2"
	"git.fiblab.net/sim/simulet-go/utils/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

// Input 输入数据
// 功能：存储仿真所需的所有输入数据
// 说明：包含地图、人员、路况、经济等各类数据，支持从文件或数据库加载
type Input struct {
	Map     *mapv2.Map
	Persons *personv2.Persons
}

// Init 下载数据
// 功能：根据配置初始化并加载所有输入数据
// 参数：config-配置对象，cacheDir-缓存目录
// 返回：加载完成的输入数据指针
// 算法说明：
// 1. 缓存检查：验证缓存目录的有效性
// 2. 数据库连接：如果配置了MongoDB则建立连接
// 3. 地图数据加载：
//   - 文件加载：从指定文件加载地图
//   - 数据库加载：从MongoDB加载地图
//
// 4. ID集合构建：构建各种地图元素的ID集合用于验证
// 5. 人员数据加载：
//   - 文件加载：支持单个或多个文件
//   - 数据库加载：从MongoDB加载并支持数据迁移
//   - 数据验证：检查位置信息的有效性
//
// 6. 人员筛选：
//   - 目标ID筛选：只加载指定ID的人员
//   - 数量限制：限制加载的人员数量
//   - 行程限制：限制每个人员的行程数量
//
// 7. 路况数据加载：并行加载路况信息
// 8. 数据验证：确保所有数据的完整性和一致性
// 说明：这是数据加载的主入口，确保仿真所需的所有数据都正确加载
func Init(config config.Config, cacheDir string) (res *Input) {
	useCache := preCheckCache(cacheDir)
	if !useCache {
		cacheDir = ""
	}

	var client *mongo.Client
	if config.Input.URI != "" {
		client = mongoutil.NewClient(config.Input.URI)
		defer client.Disconnect(context.Background())
	}

	// 初始化返回值
	res = &Input{
		Persons: &personv2.Persons{
			Persons: make([]*personv2.Person, 0),
		},
	}

	var wg sync.WaitGroup

	if config.Input.Map.File != "" {
		var m mapv2.Map
		if err := protoutil.UnmarshalFromFile(&m, config.Input.Map.File); err != nil {
			log.Panicf("failed to load map from file: %v", err)
		}
		res.Map = &m
	} else if len(config.Input.Map.Files) > 0 {
		log.Panicf("multiple map files are not supported")
	} else {
		res.Map = mustLoad[mapv2.Map](client, config.Input.Map, cacheDir, nil, nil)
	}

	ids := mapIDs{
		aoiIDs:         make(map[int32]struct{}),
		drivingLaneIDs: make(map[int32]struct{}),
		walkingLaneIDs: make(map[int32]struct{}),
		junctionIDs:    make(map[int32]struct{}),
	}
	for _, v := range res.Map.Aois {
		ids.aoiIDs[v.Id] = struct{}{}
	}
	for _, v := range res.Map.Lanes {
		switch v.Type {
		case mapv2.LaneType_LANE_TYPE_DRIVING:
			ids.drivingLaneIDs[v.Id] = struct{}{}
		case mapv2.LaneType_LANE_TYPE_WALKING:
			ids.walkingLaneIDs[v.Id] = struct{}{}
		}
	}
	for _, v := range res.Map.Junctions {
		ids.junctionIDs[v.Id] = struct{}{}
	}

	personIDs := make(map[int32]struct{})
	if config.Input.Person != nil {
		if config.Input.Person.File != "" {
			var p personv2.Persons
			if err := protoutil.UnmarshalFromFile(&p, config.Input.Person.File); err != nil {
				log.Panicf("failed to load person from file: %v", err)
			}
			res.Persons = &p
		} else if len(config.Input.Person.Files) > 0 {
			// 读取多个文件
			for _, file := range config.Input.Person.Files {
				var p personv2.Persons
				if err := protoutil.UnmarshalFromFile(&p, file); err != nil {
					log.Panicf("failed to load person from file: %v", err)
				}
				res.Persons.Persons = append(res.Persons.Persons, p.Persons...)
			}
		} else {
			res.Persons = mustLoad[personv2.Persons](client, *config.Input.Person, cacheDir, nil, func(className string, pb any, rawBson bson.Raw) error {
				person := pb.(*personv2.Person)

				// 检查数据正确性：position是否在地图中
				var badPosition *geov2.Position
				var badScheduleIndex int
				var badTripIndex int
				var badTrip *tripv2.Trip
				for i, schedule := range person.Schedules {
					for j, trip := range schedule.Trips {
						if i == 0 && j == 0 {
							if !checkPositionValid(person.Home, ids, trip.Mode) {
								badPosition = person.Home
								badScheduleIndex = i
								badTripIndex = j
								badTrip = trip
								goto INVALID
							}
						}
						if !checkPositionValid(trip.End, ids, trip.Mode) {
							badPosition = trip.End
							badScheduleIndex = i
							badTripIndex = j
							badTrip = trip
							goto INVALID
						}
					}
				}
				return nil
			INVALID:
				return fmt.Errorf("ignore person %v due to bad (position: %v, trip %d-%d: %v)", person.Id, badPosition, badScheduleIndex, badTripIndex, badTrip)
			})
		}
	}
	if config.Input.Person != nil && len(res.Persons.Persons) == 0 {
		log.Error("no valid persons to simulate, may be class=agent rather than class=person")
	}
	for _, p := range res.Persons.Persons {
		if _, ok := personIDs[p.Id]; ok {
			log.Panicf("persons have duplicated ids %d, please check data", p.Id)
		}
		personIDs[p.Id] = struct{}{}
	}

	wg.Wait()
	return
}

// mustLoad 必须加载数据（泛型函数）
// 功能：从MongoDB或缓存中加载数据，支持数据迁移和验证
// 参数：client-MongoDB客户端，inputPath-输入路径配置，cacheDir-缓存目录，classNameMapper-类名映射器，handler-数据处理函数，opts-查询选项
// 返回：加载的数据对象
// 算法说明：
// 1. 获取MongoDB集合：根据输入路径配置获取集合
// 2. 定义下载函数：如果不需要仅缓存则定义下载逻辑
// 3. 缓存加载：使用缓存机制加载数据
// 4. 错误处理：如果加载失败则panic
// 说明：提供统一的数据加载接口，支持缓存和数据库两种数据源
func mustLoad[T any, PT interface {
	proto.Message
	*T
}](
	client *mongo.Client,
	inputPath config.InputPath,
	cacheDir string,
	classNameMapper func(string) string,
	handler func(className string, pb any, rawBson bson.Raw) error,
	opts ...*options.FindOptions,
) (res PT) {
	coll := mongoutil.GetMongoColl(client, inputPath)
	var downloadFunc func() PT
	var err error
	if !inputPath.OnlyCache {
		downloadFunc = func() PT {
			pb, errs := mongoutil.DownloadPbFromMongo[T, PT](context.Background(), coll, classNameMapper, handler, opts...)
			if len(errs) > 0 {
				for _, err := range errs {
					log.Errorf("failed to download: %v", err)
				}
				log.Panicln("failed to download")
			}
			return pb
		}
	}
	log.Infof("start fetching from %s.%s", inputPath.DB, inputPath.Col)
	res, err = cache.LoadWithCache(cacheDir, inputPath, downloadFunc)
	if err != nil {
		log.Panicf("failed to load with cache: %v", err)
	}
	log.Infof("finish fetching from %s.%s", inputPath.DB, inputPath.Col)
	return
}
