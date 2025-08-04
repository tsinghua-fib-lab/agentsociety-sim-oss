package ecosim

// 默认税率档位和切分点
var (
	// DefaultBracketCutoffs 是月度收入的切分点
	// 计算方法：原始年收入 * 100 / 12 得到月度收入
	// [0, 97, 394.75, 842, 1607.25, 2041, 5103] * 100 / 12
	DefaultBracketCutoffs = []float32{0, 808.33, 3289.58, 7016.67, 13393.75, 17008.33, 42525.00}

	// DefaultBracketRates 是对应的税率
	DefaultBracketRates = []float32{0.10, 0.12, 0.22, 0.24, 0.32, 0.35, 0.37}
)
