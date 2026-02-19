package types

import "strconv"

// FormatFloat64 formats a float64 to a string with 6 decimal places.
func FormatFloat64(f float64) string {
	return strconv.FormatFloat(f, 'f', 6, 64)
}

// FormatBool formats a bool to "true" or "false".
func FormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
