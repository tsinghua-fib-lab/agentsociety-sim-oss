package ecosim

// taxesDue 计算指定收入水平的应缴税额
func taxesDue(income float32, bracketCutoffs []float32, bracketRates []float32) float32 {
	if len(bracketCutoffs) != len(bracketRates) {
		return 0
	}

	var totalTax float32

	// 从高到低处理每个税率档位
	for i := len(bracketCutoffs) - 1; i >= 0; i-- {
		if income > bracketCutoffs[i] {
			// 计算超过当前切分点的部分
			taxableIncome := income - bracketCutoffs[i]
			// 对超过部分征收相应税率
			tax := taxableIncome * bracketRates[i]
			totalTax += tax
			// 更新收入为切分点值，继续处理下一个档位
			income = bracketCutoffs[i]
		}
	}

	// 处理最低档位（如果还有剩余收入）
	// if income > 0 {
	// 	totalTax += income * bracketRates[0]
	// }

	return totalTax
}
