package marketdata

import (
	"fmt"
	"math"

	"github.com/JerrrMi/quant/internal/domain"
)

// LogReturnLast 计算最近一根相对前一根收盘的对数收益 ln(close/prior)；prior<=0 或 close<=0 时返回 error。
func LogReturnLast(window []domain.Bar) (float64, error) {
	if len(window) < 2 {
		return 0, fmt.Errorf("marketdata: log return needs at least 2 bars")
	}
	prior := window[len(window)-2].Close
	last := window[len(window)-1].Close
	if prior <= 0 || last <= 0 {
		return 0, fmt.Errorf("marketdata: non-positive close for log return")
	}
	return math.Log(last / prior), nil
}

// LogReturnsFromCloses 由收盘价序列构造相邻对数收益，长度 len(closes)-1。
func LogReturnsFromCloses(closes []float64) ([]float64, error) {
	if len(closes) < 2 {
		return nil, nil
	}
	out := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		a, b := closes[i-1], closes[i]
		if a <= 0 || b <= 0 {
			return nil, fmt.Errorf("marketdata: non-positive close at %d", i)
		}
		out = append(out, math.Log(b/a))
	}
	return out, nil
}

// NormalizeTanh01 将 x 通过 tanh 压到近似 (-1,1)，再线性映射到 [0,1]：0.5*(1+tanh(x))。
func NormalizeTanh01(x float64) float64 {
	t := math.Tanh(x)
	return 0.5 * (1 + t)
}

// NormalizeMinMax 将 x 从 [min,max] 线性映射到 [0,1]；max<=min 时返回 0.5。
func NormalizeMinMax(x, min, max float64) float64 {
	if max <= min {
		return 0.5
	}
	if x <= min {
		return 0
	}
	if x >= max {
		return 1
	}
	return (x - min) / (max - min)
}

// WindowMeanStd 计算对数收益样本的样本标准差（自由度 n-1）；n<2 返回 0,0。
func WindowMeanStd(xs []float64) (mean, std float64) {
	if len(xs) < 2 {
		return 0, 0
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	mean = sum / float64(len(xs))
	var varSum float64
	for _, v := range xs {
		d := v - mean
		varSum += d * d
	}
	std = math.Sqrt(varSum / float64(len(xs)-1))
	return mean, std
}

// WindowZScoreLast 使用窗口内对数收益的均值/标准差，将最后一根 log return 转为 Z-score；std 极小时钳制。
func WindowZScoreLast(window []domain.Bar, eps float64) (float64, error) {
	if len(window) < 3 {
		return 0, fmt.Errorf("marketdata: z-score needs at least 3 bars")
	}
	closes := make([]float64, len(window))
	for i, b := range window {
		closes[i] = b.Close
	}
	lr, err := LogReturnsFromCloses(closes)
	if err != nil {
		return 0, err
	}
	if len(lr) < 2 {
		return 0, fmt.Errorf("marketdata: insufficient log returns")
	}
	mean, std := WindowMeanStd(lr[:len(lr)-1])
	if std <= eps {
		std = eps
	}
	last := lr[len(lr)-1]
	return (last - mean) / std, nil
}
