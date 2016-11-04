GITHUB_API_SOURCE = src/poule/gh/github.go

all:
	gb build all

mocks: src/poule/test/mocks/IssuesService.go

test: mocks
	gb test all

src/poule/test/mocks/%.go: $(GITHUB_API_SOURCE)
	gb generate poule/gh

