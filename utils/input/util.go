package input

import (
	"os"

	geov2 "git.fiblab.net/sim/protos/v2/go/city/geo/v2"
	tripv2 "git.fiblab.net/sim/protos/v2/go/city/trip/v2"
)

// mapIDs 地图ID集合
// 功能：存储各种地图元素的ID集合，用于位置验证
// 说明：使用map[int32]struct{}结构实现高效的ID查找
type mapIDs struct {
	aoiIDs         map[int32]struct{} // AOI区域ID集合
	drivingLaneIDs map[int32]struct{} // 机动车道ID集合
	walkingLaneIDs map[int32]struct{} // 步行道ID集合
	junctionIDs    map[int32]struct{} // 路口ID集合
}

// checkPositionValid 检查位置有效性
// 功能：验证位置信息是否符合逻辑规则和地图约束
// 参数：pos-位置信息，ids-地图ID集合，tripMode-出行模式
// 返回：true表示位置有效，false表示位置无效
// 算法说明：
// 1. 检查位置类型：不能同时存在AOI位置和车道位置
// 2. 检查位置存在性：必须存在至少一种位置类型
// 3. 验证AOI位置：检查AOI ID是否在有效集合中
// 4. 验证车道位置：根据出行模式选择对应的车道类型进行验证
// 5. 处理未知出行模式：记录警告并默认有效
// 说明：确保位置信息与出行模式和地图数据的一致性
func checkPositionValid(pos *geov2.Position, ids mapIDs, tripMode tripv2.TripMode) bool {
	if pos.AoiPosition != nil && pos.LanePosition != nil {
		// 同时存在两个逻辑坐标
		return false
	}
	if pos.AoiPosition == nil && pos.LanePosition == nil {
		// 不存在逻辑坐标
		return false
	}
	if pos.AoiPosition != nil {
		_, ok := ids.aoiIDs[pos.AoiPosition.AoiId]
		return ok
	}
	if pos.LanePosition != nil {
		switch tripMode {
		case tripv2.TripMode_TRIP_MODE_DRIVE_ONLY:
			_, ok := ids.drivingLaneIDs[pos.LanePosition.LaneId]
			return ok
		case tripv2.TripMode_TRIP_MODE_WALK_ONLY, tripv2.TripMode_TRIP_MODE_BIKE_WALK, tripv2.TripMode_TRIP_MODE_BUS_WALK:
			_, ok := ids.walkingLaneIDs[pos.LanePosition.LaneId]
			return ok
		default:
			log.Warnf("unknown trip mode %v", tripMode)
			return true
		}
	}
	panic("impossible")
}

// preCheckCache 预检查缓存目录
// 功能：验证输入缓存目录的有效性，决定是否启用缓存功能
// 参数：cacheDir-缓存目录路径
// 返回：true表示启用缓存，false表示禁用缓存
// 算法说明：
// 1. 检查缓存目录是否为空：空则禁用缓存
// 2. 检查目录是否存在：使用os.Stat检查路径状态
// 3. 验证是否为目录：确保路径指向的是目录而不是文件
// 4. 记录日志：根据检查结果输出相应的日志信息
// 说明：确保缓存功能的正确配置，避免因无效路径导致的错误
func preCheckCache(cacheDir string) bool {
	if cacheDir == "" {
		log.Info("disable input cache")
		return false
	} else {
		if stat, err := os.Stat(cacheDir); err == nil && stat.IsDir() {
			// 文件夹存在
			log.Infof("enable input cache at %s", cacheDir)
			return true
		} else {
			log.Errorf("disable input cache because invalid dir %s (not exist or file)", cacheDir)
			return false
		}
	}
}
