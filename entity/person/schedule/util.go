package schedule

import tripv2 "git.fiblab.net/sim/protos/v2/go/city/trip/v2"

// IsDrivingTrip 检查是否是开车的出行
// 功能：判断行程是否为自驾车模式
// 参数：trip-行程信息
// 返回：true表示是自驾车出行，false表示不是
func IsDrivingTrip(trip *tripv2.Trip) bool {
	return trip.Mode == tripv2.TripMode_TRIP_MODE_DRIVE_ONLY
}

// IsWalkingTrip 检查是否是步行的出行
// 功能：判断行程是否为步行模式（包括纯步行和自行车+步行）
// 参数：trip-行程信息
// 返回：true表示是步行出行，false表示不是
func IsWalkingTrip(trip *tripv2.Trip) bool {
	return trip.Mode == tripv2.TripMode_TRIP_MODE_WALK_ONLY || trip.Mode == tripv2.TripMode_TRIP_MODE_BIKE_WALK
}
