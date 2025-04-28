package util

import (
	"math"
	"math/rand/v2"
	"path/filepath"
)

func GetRandomFromDistribution(mean, sd float64) float64 {
	return GetRandomNormalDistribution()*sd + mean
}

func GetRandomNormalDistribution() float64 {
	r1 := rand.Float64()
	r2 := rand.Float64()
	if r1 == 0 || r2 == 0 {
		return GetRandomNormalDistribution()
	}
	return math.Sqrt(-2*math.Log(r1)) * math.Cos(2*math.Pi*r2)
}

func FormatPath(path string) (string, error) {
	s, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if len(s) != 0 && s[len(s)-1] != '/' {
		s += "/"
	}
	return s, nil
}

func Sum(slice []int) (result int) {
	result = 0
	for _, v := range slice {
		result += v
	}
	return
}
