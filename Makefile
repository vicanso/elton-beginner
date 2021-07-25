.PHONY: default test test-cover dev generate hooks lint-web doc

# for dev
dev:
	air -c .air.toml	

install:
	go get entgo.io/ent/cmd/entc

generate: 
	entc generate ./schema --target ./ent
