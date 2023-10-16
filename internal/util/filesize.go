package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var suffixes = map[string]uint64{
	"":    1,
	"b":   1,
	"k":   1000,
	"kb":  1000,
	"ki":  1024,
	"kib": 1024,
	"m":   1000 * 1000,
	"mb":  1000 * 1000,
	"mi":  1024 * 1024,
	"mib": 1024 * 1024,
	"g":   1000 * 1000 * 1000,
	"gb":  1000 * 1000 * 1000,
	"gi":  1024 * 1024 * 1024,
	"gib": 1024 * 1024 * 1024,
	"t":   1000 * 1000 * 1000 * 1000,
	"tb":  1000 * 1000 * 1000 * 1000,
	"ti":  1024 * 1024 * 1024 * 1024,
	"tib": 1024 * 1024 * 1024 * 1024,
}

func GetByteSizeFromString(sizeStr string) (int64, error) {
	number, suffix, err := separateString(sizeStr)

	if err != nil {
		return 0, err
	}

	if factor, ok := suffixes[strings.ToLower(suffix)]; ok {
		size, err := strconv.ParseFloat(number, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse number '%s'", number)
		}
		return int64(float64(factor) * size), nil
	} else {
		return 0, fmt.Errorf("suffix '%s' not recognized", suffix)
	}
}

func GetStringFromByteSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KiB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MiB", float64(size)/(1024*1024))
	} else if size < 1024*1024*1024*1024 {
		return fmt.Sprintf("%.2f GiB", float64(size)/(1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2f TiB", float64(size)/(1024*1024*1024*1024))
	}
}

func separateString(s string) (number string, suffix string, err error) {
	r := regexp.MustCompile(`^([\d\.]+)\s*([a-zA-Z]*)$`)
	matches := r.FindStringSubmatch(s)

	if len(matches) != 3 {
		return "", "", fmt.Errorf("string format not recognized")
	}

	return matches[1], matches[2], nil
}
