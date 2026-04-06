package w

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/pulsoats/core/domain/detect"
	"github.com/pulsoats/core/domain/identity"
	"github.com/pulsoats/core/domain/market"
	"github.com/pulsoats/core/lib/units"
)

// Detect - поиск паттерна в переданном окне, возвращает сигнал при успешном поиске.
// Сигнал выставляется по последней свече окна.
// "Не пробила бай" — по High.
func (d *Detector) Detect(ctx context.Context, window []market.Candle, fees market.TakerMakerFees) (detect.Signal, bool, error) {
	var zero detect.Signal

	if err := ctx.Err(); err != nil {
		return zero, false, err
	}

	opts := d.opts

	minsIdxs, err := findLocalMins(ctx, window)
	if err != nil {
		return zero, false, err
	}
	if len(minsIdxs) < 2 {
		return zero, false, nil
	}

	// 1) Поиск наименьшего из найденных локальных минимумов
	lowestMinIdx := minsIdxs[0]
	lowestMinVal := window[lowestMinIdx].Close
	for i := 1; i < len(minsIdxs); i++ {
		if err := ctx.Err(); err != nil {
			return zero, false, err
		}
		idx := minsIdxs[i]
		v := window[idx].Close
		if v < lowestMinVal {
			lowestMinIdx = idx
			lowestMinVal = v
		}
	}

	// 2) Отбор минимумов по LocalMinsDeviation относительно lowestMinVal
	acceptedMinsIdxs := make([]int, 0, len(minsIdxs))
	for _, idx := range minsIdxs {
		if err := ctx.Err(); err != nil {
			return zero, false, err
		}
		if idx == lowestMinIdx {
			acceptedMinsIdxs = append(acceptedMinsIdxs, idx)
			continue
		}

		v := window[idx].Close
		if idx < lowestMinIdx {
			// нормализация относительно lowestMinVal
			if AbsInt64(v-lowestMinVal)*units.PPM <= opts.LocalMinsDeviation*lowestMinVal {
				acceptedMinsIdxs = append(acceptedMinsIdxs, idx)
			}
			continue
		}
		// idx > lowestMinIdx: нормализация относительно v (как у тебя было)
		if AbsInt64(v-lowestMinVal)*units.PPM <= opts.LocalMinsDeviation*v {
			acceptedMinsIdxs = append(acceptedMinsIdxs, idx)
		}
	}

	if len(acceptedMinsIdxs) < 2 {
		return zero, false, nil
	}

	leftMinIdx := acceptedMinsIdxs[0]
	rightMinIdx := acceptedMinsIdxs[len(acceptedMinsIdxs)-1]

	// Между минимумами должна быть хотя бы 1 свеча (локальный максимум)
	if rightMinIdx-leftMinIdx < 2 {
		return zero, false, nil
	}

	leftMinVal := window[leftMinIdx].Close

	// 3) Максимум (buy) между минимумами — по Close
	maxIndex := leftMinIdx + 1
	maxVal := window[maxIndex].Close
	for i := leftMinIdx + 2; i < rightMinIdx; i++ {
		if err := ctx.Err(); err != nil {
			return zero, false, err
		}
		if window[i].Close > maxVal {
			maxVal = window[i].Close
			maxIndex = i
		}
	}

	// 4) Амплитуда между левым минимумом и максимумом должна быть достаточной (по Close)
	if (maxVal-leftMinVal)*units.PPM < leftMinVal*opts.MinMaxDeviation {
		return zero, false, nil
	}

	// 5) Поиск свечи с наибольшим Close > maxVal слева от левого минимума
	leftMaxVal := int64(math.MinInt64)
	for i := 0; i < leftMinIdx; i++ {
		if err := ctx.Err(); err != nil {
			return zero, false, err
		}
		if window[i].Close >= leftMaxVal && window[i].Close > maxVal {
			leftMaxVal = window[i].Close
		}
	}
	if leftMaxVal == int64(math.MinInt64) {
		// нет "левой вершины" > maxVal => тренд/геометрия не подтверждены
		return zero, false, nil
	}

	// Базовый TakeProfit от "левой вершины"
	tpValue := leftMaxVal - ((leftMaxVal-maxVal)*opts.TakeProfitRatio)/units.PPM

	// 6) Проверка комиссий/спреда: TP должен перекрывать издержки (как у тебя)
	minTP := (maxVal*(fees.TakerFeeRate+market.SpreadPPM)+tpValue*fees.MakerFeeRate)/units.PPM + maxVal
	if tpValue < minTP {
		return zero, false, nil
	}

	// 7) Объемы
	if len(window) >= 7 {
		var sum int64
		start := len(window) - 2
		end := len(window) - 7

		stdData := make([]int64, 0, 6)
		for i := start; i >= end; i-- {
			sum += window[i].Volume
			stdData = append(stdData, window[i].Volume)
		}

		// средний объем за 6 свечей перед сигнальной
		avgVol6 := sum / 6
		if avgVol6 == 0 {
			return zero, false, nil
		}

		// стандартное отклонение по тем же 6 свечам
		stdVol6 := standardDeviation(stdData)

		// последняя свеча перед сигналом
		lastVol := window[len(window)-2].Volume

		// все коэффициенты в PPM
		noSpike := lastVol < (avgVol6*opts.VolumeSpikeMultiplier)/units.PPM
		volCV := (stdVol6 * units.PPM) / avgVol6
		stableVolume := volCV < opts.VolumeCVThreshold

		// отбрасываем плохой сигнал
		if !noSpike || !stableVolume {
			return zero, false, nil
		}
	}

	// Stop Loss
	// slValue := lowestMinVal * opts.StopLossRatio/units.PPM
	slValue := lowestMinVal - (leftMaxVal-maxVal)*opts.StopLossRatio/units.PPM

	// Сигнал выставляем на последней свече окна (лайв-точка)
	lastIdx := len(window) - 1

	id, _ := uuid.NewV7()

	extremes := []market.Candle{window[leftMinIdx], window[maxIndex], window[rightMinIdx]}

	return detect.Signal{
		ID:              id,
		Detector:        d.Code(),
		Time:            window[lastIdx].Time,
		Value:           window[lastIdx].Close, // сигнал по текущей (последней) свече окна
		BuyValue:        maxVal,
		TakeProfitValue: tpValue,
		StopLossValue:   slValue,
		Extremes:        extremes,
		Fingerprint:     identity.MakeFingerprint(fmt.Sprintf("%s|%s|%v", d.Code(), d.label, window[maxIndex].Time)),
	}, true, nil
}

// findLocalMins - поиск локальных минимумов по Close.
// Внутри окна: правильный локальный минимум (ниже чем точки слева и справа).
// Последняя точка: допускается как кандидат, если это новый минимум относительно всего слева (по Close).
func findLocalMins(ctx context.Context, window []market.Candle) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	n := len(window)

	localMinsIdxs := make([]int, 0, n/2)

	// 1) поиск всех локальных минимумов в окне
	for i := 1; i < n-1; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if window[i-1].Close > window[i].Close && window[i].Close < window[i+1].Close {
			localMinsIdxs = append(localMinsIdxs, i)
		}
	}

	// 2) последняя точка как кандидат правого лок минимума:
	// будет учтена только если меньше всех точек, начиная с левого локального минимума
	last := n - 1
	if len(localMinsIdxs) > 0 {
		prevMinIdx := localMinsIdxs[len(localMinsIdxs)-1]

		// нужна хотя бы одна свеча "между" prevMinIdx и last, иначе это просто сосед
		if last-prevMinIdx >= 2 {
			lastClose := window[last].Close
			ok := true

			// проверяем только участок между prevMinIdx и last (не включая их)
			for i := prevMinIdx + 1; i < last; i++ {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				// правило: lastClose должен быть ниже любых свечей по close справа от prevMinIdx
				if window[i].Close <= lastClose {
					ok = false
					break
				}
			}

			if ok {
				localMinsIdxs = append(localMinsIdxs, last)
			}
		}
	}

	if len(localMinsIdxs) < 2 {
		return nil, nil
	}
	return localMinsIdxs, nil
}

// AbsInt64 - модуль для типа int64
func AbsInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func standardDeviation(data []int64) int64 {
	n := len(data)
	if n < 2 {
		return 0
	}

	var sum int64
	for _, v := range data {
		sum += v
	}

	mean := sum / int64(n)

	var sumSq float64
	for _, v := range data {
		diff := float64(v - mean)
		sumSq += diff * diff
	}

	variance := sumSq / float64(n)

	return int64(math.Sqrt(variance))
}
