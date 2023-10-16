package util

import (
	"testing"
)

func TestGetByteSizeFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		err      bool
	}{
		{"1000", 1000, false},
		{"10B", 10, false},
		{"10K", 10000, false},
		{"10Ki", 10240, false},
		{"10KiB", 10240, false},
		{"10M", 10000000, false},
		{"10Mi", 10485760, false},
		{"10MiB", 10485760, false},
		{"10G", 10000000000, false},
		{"10Gi", 10737418240, false},
		{"10GiB", 10737418240, false},
		{"10T", 10000000000000, false},
		{"10Ti", 10995116277760, false},
		{"10TiB", 10995116277760, false},
		{"invalid", 0, true},
		{"10X", 0, true},
	}

	for _, test := range tests {
		result, err := GetByteSizeFromString(test.input)
		if test.err {
			if err == nil {
				t.Errorf("Expected an error for input %s but got none", test.input)
			}
		} else {
			if result != test.expected {
				t.Errorf("Expected %d for input %s but got %d", test.expected, test.input, result)
			}
		}
	}
}

func TestGetStringFromByteSize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.00 KiB"},
		{1025, "1.00 KiB"},
		{1100, "1.07 KiB"},
		{1048575, "1024.00 KiB"},
		{1048576, "1.00 MiB"},
		{1048577, "1.00 MiB"},
		{1048576*1024 - 1, "1024.00 MiB"},
		{1048576 * 1024, "1.00 GiB"},
		{1048576 * 1024 * 1023, "1023.00 GiB"},
		{1048576*1024*1024 - 1, "1024.00 GiB"},
		{1048576 * 1024 * 1024, "1.00 TiB"},
	}

	for _, test := range tests {
		result := GetStringFromByteSize(test.input)
		if result != test.expected {
			t.Errorf("Expected %s for input %d but got %s", test.expected, test.input, result)
		}
	}
}
