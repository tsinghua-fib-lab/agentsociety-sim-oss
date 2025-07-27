package config

// InputPath 指定输入数据来源的配置（MongoDB、文件系统）
// 功能：定义数据输入路径的配置结构，支持多种数据源
// 说明：支持MongoDB数据库和文件系统两种数据源，支持缓存机制
type InputPath struct {
	DB        string   `yaml:"db"`                   // 数据库名
	Col       string   `yaml:"col"`                  // 集合名
	Cache     string   `yaml:"cache,omitempty"`      // 缓存文件名，为空则采用默认路径{db}.{col}.pb
	OnlyCache bool     `yaml:"only_cache,omitempty"` // 只从缓存中获取
	File      string   `yaml:"file,omitempty"`       // 文件路径（优先级高于MongoDB）
	Files     []string `yaml:"files,omitempty"`      // 文件路径列表（优先级高于MongoDB）
}

// GetDb 获取数据库名
// 功能：返回配置的数据库名称
// 返回：数据库名称字符串
func (p InputPath) GetDb() string {
	return p.DB
}

// GetColl 获取集合名
// 功能：返回配置的集合名称
// 返回：集合名称字符串
func (p InputPath) GetColl() string {
	return p.Col
}

// GetCachePath 获取缓存文件路径
// 功能：返回缓存文件的完整路径
// 返回：缓存文件路径字符串
// 算法说明：
// 1. 如果指定了缓存路径，直接返回
// 2. 否则使用默认命名规则：{数据库名}.{集合名}.pb
// 说明：提供统一的缓存路径获取接口
func (p InputPath) GetCachePath() string {
	if p.Cache != "" {
		return p.Cache
	}
	return p.DB + "." + p.Col + ".pb"
}

// Input 指定模拟器所有输入数据的配置项
// 功能：定义仿真系统的所有输入数据配置
// 说明：包含地图、人员、路况等各类输入数据的配置
type Input struct {
	URI    string     `yaml:"uri"`              // MongoDB连接字符串
	Map    InputPath  `yaml:"map"`              // 地图
	Person *InputPath `yaml:"person,omitempty"` // 人员
}

// ControlStep 指定模拟器模拟时间范围和间隔的配置项
// 功能：定义仿真时间控制参数
// 说明：控制仿真的时间范围、步长和精度
type ControlStep struct {
	Start    int32   `yaml:"start"`    // 开始步数
	Total    int32   `yaml:"total"`    // 总步数
	Interval float64 `yaml:"interval"` // 每步的时间间隔
}

// Control 模拟器控制配置
// 功能：定义仿真系统的核心控制参数
// 说明：包含时间控制、区域范围、功能开关等核心配置
type Control struct {
	Step             ControlStep `yaml:"step"`
	PreferFixedLight bool        `yaml:"prefer_fixed_light,omitempty"` // 优先使用固定相位信控，如果不存在则使用最大
}

// Config YAML配置文件的根结构
// 功能：定义整个仿真系统的配置结构
// 说明：包含输入、控制、输出等所有配置项
type Config struct {
	Input   Input   `yaml:"input"`   // 输入
	Control Control `yaml:"control"` // 模拟过程控制
}
