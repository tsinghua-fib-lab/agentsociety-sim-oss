package person

import "github.com/sirupsen/logrus"

// log 人员模块的日志记录器
// 功能：为person模块提供统一的日志记录功能
// 说明：使用logrus库，并添加"module"字段标识为"person"模块
var log = logrus.WithField("module", "person")
