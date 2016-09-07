package gh

import (
	"log"
	"strings"
)

func GetRepository(repository string) (string, string) {
	s := strings.SplitN(repository, "/", 2)
	if len(s) != 2 {
		log.Fatalf("Invalid repository specification %q", repository)
	}
	return s[0], s[1]
}
