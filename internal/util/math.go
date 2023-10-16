package util

import (
	"github.com/montanaflynn/stats"
	"github.com/rs/zerolog/log"
)

func GetMinFloat64(slice []float64) float64 {
	if len(slice) == 0 {
		return 0
	}
	min := slice[0]
	for _, value := range slice {
		if value < min {
			min = value
		}
	}
	return min
}

func GetMaxFloat64(slice []float64) float64 {
	if len(slice) == 0 {
		return 0
	}
	max := slice[0]
	for _, value := range slice {
		if value > max {
			max = value
		}
	}
	return max
}

func GetPercentileFloat64(slice []float64, p float64) float64 {
	if len(slice) == 0 {
		return 0
	}
	percentile, err := stats.Percentile(slice, p)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get percentile")
		return 0
	}
	return percentile
}

func GetStdDevFloat64(slice []float64) float64 {
	if len(slice) == 0 {
		return 0
	}
	stdDev, err := stats.StandardDeviation(slice)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get standard deviation")
		return 0
	}
	return stdDev
}

func GetMean(slice []float64) float64 {
	if len(slice) == 0 {
		return 0
	}
	average, err := stats.Mean(slice)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get average")
		return 0
	}
	return average
}
