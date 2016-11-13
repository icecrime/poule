package settings

import (
	"fmt"
	"time"
)

// ExtDuration is an extension of time.Duration to support units longer than hours.
type ExtDuration struct {
	Quantity int64
	Unit     rune
}

// ParseExtDuration parses an ExtDuration from a string.
func ParseExtDuration(value string) (ExtDuration, error) {
	e := ExtDuration{}
	if n, err := fmt.Sscanf(value, "%d%c", &e.Quantity, &e.Unit); n != 2 {
		return e, fmt.Errorf("Invalid value %q for duration", value)
	} else if err != nil {
		return e, fmt.Errorf("Invalid value %q for duration: %v", value, err)
	}
	switch e.Unit {
	case 'd', 'D', 'w', 'W', 'm', 'M', 'y', 'Y':
		break
	default:
		return e, fmt.Errorf("Invalid unit \"%c\" for threshold", e.Unit)
	}
	return e, nil
}

// Duration returns the value as a time.Duration.
func (e ExtDuration) Duration() time.Duration {
	day := 24 * time.Hour
	switch e.Unit {
	case 'd', 'D':
		return time.Duration(e.Quantity) * day
	case 'w', 'W':
		return time.Duration(e.Quantity) * 7 * day
	case 'm', 'M':
		return time.Duration(e.Quantity) * 31 * day
	case 'y', 'Y':
		return time.Duration(e.Quantity) * 356 * day
	default:
		panic("invalid duration unit")
	}
}

// String returns a string representation of the ExtDuration.
func (e ExtDuration) String() string {
	switch e.Unit {
	case 'd', 'D':
		return fmt.Sprintf("%d %s", e.Quantity, pluralize(e.Quantity, "day"))
	case 'w', 'W':
		return fmt.Sprintf("%d %s", e.Quantity, pluralize(e.Quantity, "week"))
	case 'm', 'M':
		return fmt.Sprintf("%d %s", e.Quantity, pluralize(e.Quantity, "month"))
	case 'y', 'Y':
		return fmt.Sprintf("%d %s", e.Quantity, pluralize(e.Quantity, "year"))
	default:
		panic("invalid duration unit")
	}
}

func pluralize(count int64, value string) string {
	if count == 1 {
		return value
	}
	return value + "s"
}
