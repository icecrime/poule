GITHUB_API_SOURCE = src/poule/gh/github.go
PREFIX := /usr/local

all:
	gb build all

install: 
	install -m 755 bin/poule ${PREFIX}/bin/poule

mocks: src/poule/test/mocks/IssuesService.go

test: mocks
	gb test all

src/poule/test/mocks/%.go: $(GITHUB_API_SOURCE)
	gb generate poule/gh

