package crontab

import (
	"fmt"
	"strconv"
	"strings"
)

type FieldKind int

const (
	FieldMinute FieldKind = iota
	FieldHour
	FieldDom
	FieldMonth
	FieldDow
)

func DescribeField(kind FieldKind, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "empty"
	}
	if value == "*" {
		switch kind {
		case FieldMinute:
			return "every minute"
		case FieldHour:
			return "every hour"
		case FieldDom:
			return "every day"
		case FieldMonth:
			return "every month"
		case FieldDow:
			return "every day of week"
		}
	}
	if strings.HasPrefix(value, "*/") {
		step := strings.TrimPrefix(value, "*/")
		if _, err := strconv.Atoi(step); err == nil {
			switch kind {
			case FieldMinute:
				return "every " + step + " minutes"
			case FieldHour:
				return "every " + step + " hours"
			case FieldDom:
				return "every " + step + " days"
			case FieldMonth:
				return "every " + step + " months"
			case FieldDow:
				return "every " + step + " days of week"
			}
		}
	}
	if strings.Contains(value, ",") {
		return labelFor(kind) + " " + value
	}
	if strings.Contains(value, "-") {
		return labelFor(kind) + " range " + value
	}
	if n, err := strconv.Atoi(value); err == nil {
		switch kind {
		case FieldMinute:
			return fmt.Sprintf("at minute %d", n)
		case FieldHour:
			return fmt.Sprintf("at hour %d", n)
		case FieldDom:
			return fmt.Sprintf("on day %d of the month", n)
		case FieldMonth:
			return "in " + monthName(n)
		case FieldDow:
			return "on " + dayName(n)
		}
	}
	return value
}

func labelFor(k FieldKind) string {
	switch k {
	case FieldMinute:
		return "minutes"
	case FieldHour:
		return "hours"
	case FieldDom:
		return "days"
	case FieldMonth:
		return "months"
	case FieldDow:
		return "days of week"
	}
	return ""
}

func monthName(n int) string {
	names := []string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	if n >= 1 && n <= 12 {
		return names[n]
	}
	return strconv.Itoa(n)
}

func dayName(n int) string {
	names := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	if n >= 0 && n <= 6 {
		return names[n]
	}
	if n == 7 {
		return "Sunday"
	}
	return strconv.Itoa(n)
}
