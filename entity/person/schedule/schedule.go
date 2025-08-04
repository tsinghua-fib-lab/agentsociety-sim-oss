package schedule

import (
	"fmt"

	"git.fiblab.net/general/common/v2/mathutil"
	"git.fiblab.net/general/common/v2/protoutil"
	geov2 "git.fiblab.net/sim/protos/v2/go/city/geo/v2"
	mapv2 "git.fiblab.net/sim/protos/v2/go/city/map/v2"
	tripv2 "git.fiblab.net/sim/protos/v2/go/city/trip/v2"
	"github.com/samber/lo"
	"github.com/tsinghua-fib-lab/moss-agentsociety-go/entity"
)

// Schedule 时刻表
// 功能：管理人员的出行计划，包含多个行程安排和循环逻辑
type Schedule struct {
	ctx entity.ITaskContext

	origin          []*tripv2.Schedule // 原始时刻表（Forever模式下重置后恢复到这个状态）
	base            []*tripv2.Schedule // 时刻表
	ScheduleIndex   int32              // 当前schedule下标
	TripIndex       int32              // 当前trip下标
	loopCount       int32              // schedule循环计数器
	lastTripEndTime float64            // 上次trip结束时间
}

// NewSchedule 创建一个时刻表实例
// 功能：初始化时刻表，克隆原始数据以避免修改
// 参数：ctx-任务上下文，origin-原始时刻表数据
// 返回：初始化完成的时刻表实例
func NewSchedule(ctx entity.ITaskContext, origin []*tripv2.Schedule) *Schedule {
	return &Schedule{
		ctx: ctx,
		origin: lo.Map(origin, func(s *tripv2.Schedule, _ int) *tripv2.Schedule {
			return protoutil.Clone(s)
		}),
		base: make([]*tripv2.Schedule, 0),
	}
}

// Base 获取时刻表
// 功能：返回当前使用的时刻表数据
// 返回：时刻表数据列表
func (s *Schedule) Base() []*tripv2.Schedule {
	return s.base
}

// NextTrip 进入下一个trip，返回是否成功（是否还有trip）
// 功能：推进到下一个行程，处理循环和等待时间逻辑
// 参数：time-当前时间
// 返回：true表示还有行程，false表示所有行程已完成
// 算法说明：
// 1. 检查当前schedule是否还有trip
// 2. 更新trip索引和循环计数
// 3. 处理schedule间的等待时间或出发时间
// 4. 当所有schedule完成时返回false
func (s *Schedule) NextTrip(time float64) bool {
	if len(s.base) == 0 {
		return false
	}
	schedule := s.base[s.ScheduleIndex]
	s.lastTripEndTime = time
	if s.TripIndex++; s.TripIndex == int32(len(schedule.Trips)) {
		s.TripIndex = 0
		if s.loopCount++; schedule.LoopCount > 0 && s.loopCount >= schedule.LoopCount {
			s.loopCount = 0
			if s.ScheduleIndex++; s.ScheduleIndex == int32(len(s.base)) {
				s.base = make([]*tripv2.Schedule, 0)
				s.ScheduleIndex = 0
				return false
			} else {
				if waitTime := s.base[s.ScheduleIndex].WaitTime; waitTime != nil {
					s.lastTripEndTime += *waitTime
				} else if departureTime := s.base[s.ScheduleIndex].DepartureTime; departureTime != nil {
					s.lastTripEndTime = *departureTime
				}
			}
		}
	}
	return true
}

// GetTrip 获取当前trip
// 功能：返回当前正在执行的行程
// 返回：当前行程，如果没有则返回nil
func (s *Schedule) GetTrip() *tripv2.Trip {
	if s.ScheduleIndex >= int32(len(s.base)) {
		return nil
	}
	trips := s.base[s.ScheduleIndex].Trips
	if s.TripIndex >= int32(len(trips)) {
		return nil
	}
	return trips[s.TripIndex]
}

// Set 设置时刻表
// 功能：设置新的时刻表，验证行程的有效性
// 参数：base-新的时刻表数据，time-当前时间
// 说明：过滤无效的行程，重置索引和计数器
func (s *Schedule) Set(base []*tripv2.Schedule, time float64) {
	// 错误检查
	okBase := make([]*tripv2.Schedule, 0, len(base))
	for _, schedule := range base {
		okTrips := make([]*tripv2.Trip, 0, len(schedule.Trips))
		for _, trip := range schedule.Trips {
			switch trip.Mode {
			case tripv2.TripMode_TRIP_MODE_DRIVE_ONLY:
				if err := s.checkDrivingPositionOk(trip.End); err != nil {
					log.Warnf("invalid trip %v, %v, skip it", trip, err)
					continue
				}
			case tripv2.TripMode_TRIP_MODE_WALK_ONLY, tripv2.TripMode_TRIP_MODE_BIKE_WALK:
				if err := s.checkWalkingPositionOk(trip.End); err != nil {
					log.Warnf("invalid trip %v, %v, skip it", trip, err)
					continue
				}
			}
			okTrips = append(okTrips, trip)
		}
		if len(okTrips) != 0 {
			schedule.Trips = okTrips
			okBase = append(okBase, schedule)
		}
	}

	s.base = okBase
	s.ScheduleIndex, s.TripIndex, s.loopCount = 0, 0, 0
	if len(okBase) == 0 {
		s.lastTripEndTime = time
		return
	}
	if lastDepartureTime := okBase[0].DepartureTime; lastDepartureTime != nil {
		s.lastTripEndTime = *lastDepartureTime
	} else if waitTime := okBase[0].WaitTime; waitTime != nil {
		s.lastTripEndTime = time + *waitTime
	} else {
		s.lastTripEndTime = time
	}
}

// Empty 判断时刻表是否为空
// 功能：检查时刻表是否还有行程
// 返回：true表示空，false表示还有行程
func (s *Schedule) Empty() bool {
	return len(s.base) == 0
}

// GetDepartureTime 获取当前trip的出发时间
// 功能：计算当前行程的出发时间
// 返回：出发时间，如果没有行程则返回无穷大
// 说明：优先使用行程的出发时间，其次使用等待时间
func (s *Schedule) GetDepartureTime() float64 {
	if len(s.base) == 0 {
		//没有日程则返回∞
		return mathutil.INF
	}
	trip := s.GetTrip()
	if departureTime := trip.DepartureTime; departureTime != nil {
		if s.loopCount != 0 {
			log.Warn("departure time used in loop")
		}
		return *departureTime
	}
	if waitTime := trip.WaitTime; waitTime != nil {
		return s.lastTripEndTime + *waitTime
	} else {
		return s.lastTripEndTime
	}
}

// checkDrivingPositionOk 检查驾驶行程的终点位置是否有效
// 功能：验证驾驶行程终点是否为有效的驾驶位置
// 参数：pos-位置信息
// 返回：错误信息，nil表示有效
// 说明：检查AOI是否有驾驶车道，或车道是否为驾驶类型
func (s *Schedule) checkDrivingPositionOk(pos *geov2.Position) error {
	if pos.AoiPosition != nil {
		// 如果这个AOI没有driving gate，那么这个trip就是无效的
		aoiID := pos.AoiPosition.AoiId
		aoi := s.ctx.AoiManager().Get(aoiID)
		if aoi == nil {
			return fmt.Errorf("no such aoi %d driving gate, skip it", aoiID)
		}
		if len(aoi.DrivingLanes()) == 0 {
			return fmt.Errorf("such aoi %d has no driving lanes, skip it", aoiID)
		}
	} else if pos.LanePosition != nil {
		// lane要存在，且是drivingLane，否则无效
		laneID := pos.LanePosition.LaneId
		lane := s.ctx.LaneManager().Get(laneID)
		if lane == nil {
			return fmt.Errorf("no such lane %d, skip it", laneID)
		}
		if lane.Type() != mapv2.LaneType_LANE_TYPE_DRIVING {
			return fmt.Errorf("lane %d is not driving lane, skip it", laneID)
		}
	} else {
		return fmt.Errorf("no end position, skip it")
	}
	return nil
}

// checkWalkingPositionOk 检查步行行程的终点位置是否有效
// 功能：验证步行行程终点是否为有效的步行位置
// 参数：pos-位置信息
// 返回：错误信息，nil表示有效
// 说明：检查AOI是否有步行车道，或车道是否为步行类型
func (s *Schedule) checkWalkingPositionOk(pos *geov2.Position) error {
	if pos.AoiPosition != nil {
		// 如果这个AOI没有walking gate，那么这个trip就是无效的
		aoiID := pos.AoiPosition.AoiId
		aoi := s.ctx.AoiManager().Get(aoiID)
		if aoi == nil {
			return fmt.Errorf("no such aoi %d walking gate, skip it", aoiID)
		}
		if len(aoi.WalkingLanes()) == 0 {
			return fmt.Errorf("such aoi %d has no walking lanes, skip it", aoiID)
		}
	} else if pos.LanePosition != nil {
		// lane要存在，且是walkingLane，否则无效
		laneID := pos.LanePosition.LaneId
		lane := s.ctx.LaneManager().Get(laneID)
		if lane == nil {
			return fmt.Errorf("no such lane %d, skip it", laneID)
		}
		if lane.Type() != mapv2.LaneType_LANE_TYPE_WALKING {
			return fmt.Errorf("lane %d is not walking lane, skip it", laneID)
		}
	} else {
		return fmt.Errorf("no end position, skip it")
	}
	return nil
}
