package settings

import (
	"strings"

	"poule/utils"

	"github.com/urfave/cli"
)

const filterFlagName = "filter"

var FilteringFlag = cli.StringSliceFlag{
	Name:  filterFlagName,
	Usage: "filter based on item attributes",
}

func ParseCliFilters(c *cli.Context) ([]*utils.Filter, error) {
	value, err := NewMultiValuedKeysFromSlice(c.StringSlice(filterFlagName))
	if err != nil {
		return nil, err
	}
	return ParseConfigurationFilters(value)
}

func ParseConfigurationFilters(values map[string][]string) ([]*utils.Filter, error) {
	filters := []*utils.Filter{}
	for filterType, value := range values {
		filter, err := utils.MakeFilter(filterType, strings.Join(value, ","))
		if err != nil {
			return []*utils.Filter{}, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}
