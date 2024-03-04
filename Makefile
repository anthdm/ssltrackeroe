build:
	@go build -o bin/app

run: build
	@./bin/app

pulser:
	@go build -o bin/pulser cmd/pulser/main.go
	@./bin/pulser

mig:
	@migrate -path ./db/migrations -database postgresql://postgres:MasterElite2288@db.hfyqevvpddvobzplpkjj.supabase.co:5432/postgres up

drop:
	@migrate -path ./db/migrations -database postgresql://postgres:MasterElite2288@db.hfyqevvpddvobzplpkjj.supabase.co:5432/postgres down

cmig:
	@migrate create -ext sql -dir ./db/migrations $(filter-out $@,$(MAKECMDGOALS))

stripe:
	@stripe listen --forward-to http://localhost:3000/stripe/webhook