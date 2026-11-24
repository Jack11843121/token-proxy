// Package billing 负责模型定价与额度（美元等价单位）计算。
package billing

import "database/sql"

// Price 返回模型每 100 万 tokens 的输入/输出单价（单位：额度）。
// 若该模型未在后台定价，返回兜底默认价。
func Price(db *sql.DB, modelName string) (inputPer1M, outputPer1M float64) {
	var in, out float64
	err := db.QueryRow(`SELECT price_input, price_output FROM models WHERE name = ? AND status = 1`, modelName).Scan(&in, &out)
	if err != nil {
		return 2.0, 8.0
	}
	return in, out
}

// Spend 计算单次调用消耗的额度（= 输入 tokens 单价 + 输出 tokens 单价）。
func Spend(inputPer1M, outputPer1M float64, promptTokens, completionTokens int) float64 {
	return float64(promptTokens)/1e6*inputPer1M + float64(completionTokens)/1e6*outputPer1M
}