package settings

import (
	"fmt"
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
	value := map[string][]string{}
	for _, filter := range c.StringSlice(filterFlagName) {
		s := strings.SplitN(filter, ":", 2)
		if len(s) != 2 {
			return nil, fmt.Errorf("invalid filter format %q", filter)
		}
		value[s[0]] = strings.Split(s[1], ",")
	}
	return ParseConfigurationFilters(value)
}

func ParseConfigurationFilters(value map[string][]string) ([]*utils.Filter, error) {
	filters := []*utils.Filter{}
	for filterType, value := range value {
		filter, err := utils.MakeFilter(filterType, strings.Join(value, ","))
		if err != nil {
			return []*utils.Filter{}, err
		}
		filters = append(filters, filter)
	}
	return filters, nil
}
