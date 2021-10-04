.PHONY: default test test-cover dev generate hooks lint-web doc

# for dev
dev:
	air -c .air.toml	

install:
	go get -d entgo.io/ent/cmd/entc

generate: 
	rm -rf ./ent
	go run entgo.io/ent/cmd/ent generate ./schema --template ./template --target ./ent
