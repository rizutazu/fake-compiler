package util

import (
	"fmt"
	"golang.org/x/term"
	"math"
	"math/rand/v2"
	"path/filepath"
	"sync"
)

var lock *sync.Mutex = new(sync.Mutex)

func GetRandomFromDistribution(mean, sd float64) float64 {
	return GetRandomNormalDistribution()*sd + mean
}

func GetRandomNormalDistribution() float64 {
	lock.Lock()
	r1 := rand.Float64()
	r2 := rand.Float64()
	if r1 == 0 || r2 == 0 {
		lock.Unlock()
		return GetRandomNormalDistribution()
	}
	lock.Unlock()
	return math.Sqrt(-2*math.Log(r1)) * math.Cos(2*math.Pi*r2)
}

func GetRandomUniformDistribution(lower, upper float64) float64 {
	if lower > upper {
		return GetRandomUniformDistribution(upper, lower)
	} else if lower == upper {
		return lower
	} else {
		lock.Lock()
		r1 := rand.Float64()
		lock.Unlock()
		r1 *= upper - lower
		r1 += lower
		return r1
	}
}

func FormatPathWithSlashEnding(path string) (string, error) {
	s, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if len(s) != 0 && s[len(s)-1] != '/' {
		s += "/"
	}
	return s, nil
}

func FormatPathWithoutSlashEnding(path string) (string, error) {
	return filepath.Abs(path)
}

func Sum(slice []int) (result int) {
	result = 0
	for _, v := range slice {
		result += v
	}
	return
}

func PrintSomethingAtBottom(content string) {
	_, height, err := term.GetSize(0)
	if err != nil {
		return
	}
	// reference: https://github.com/elulcao/progress-bar/blob/main/cmd/progress-bar.go
	fmt.Print("\x1B7")       // Save the cursor position
	fmt.Print("\x1B[2K")     // Erase the entire line
	fmt.Print("\x1B[0J")     // Erase from cursor to end of screen
	fmt.Print("\x1B[?47h")   // Save screen
	fmt.Print("\x1B[1J")     // Erase from cursor to beginning of screen
	fmt.Print("\x1B[?47l")   // Restore screen
	defer fmt.Print("\x1B8") // Restore the cursor position util new size is calculated

	fmt.Printf("\x1B[%d;%dH", height, 0)

	fmt.Print(content)
}
