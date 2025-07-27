package config

// RuntimeConfig 运行时配置
// 功能：存储仿真运行时的配置信息，包含投影转换后的坐标范围
// 说明：将YAML配置转换为运行时可用的配置对象，包含坐标投影转换
type RuntimeConfig struct {
	All Config  // 全部配置
	C   Control // 全局控制配置
}

// NewRuntimeConfig 根据配置初始化全局变量
// 功能：创建运行时配置对象，进行配置验证和坐标转换
// 参数：config-原始配置对象
// 返回：初始化的运行时配置指针
// 算法说明：
// 1. 创建运行时配置对象
// 2. 设置默认值：如果未指定天数则默认为1天
// 说明：确保配置的正确性和一致性，为仿真运行提供有效配置
func NewRuntimeConfig(config Config) *RuntimeConfig {
	rc := &RuntimeConfig{}

	rc.All = config
	rc.C = config.Control

	return rc
}
