package utils

import "time"

func MustParseTime(date string, format string) time.Time {
	t, err := time.Parse(format, date)
	if err != nil {
		panic(err)
	}
	return t
}
