# DB_URL=postgresql://user:password@host:port/db_name?sslmode=disable
# sslmode=disable because the local docker postgres has no SSL set up
DB_URL=postgresql://root:secret@localhost:5433/bmc?sslmode=disable

# docker run --name {container_name} -p(for port mapping) our_port:port_that_docker_uses -e(to set env variables) POSTGRES_USER=username -e POSTGRES_PASSWORD=dbpassword -d(runs in detached mode) name_of_the_image
postgres:
	docker run --name postgres17 -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:17-alpine

# -it runs it in interactive mode
# docker exec -it {container_name} createdb --username=db_user --owner=db_owner db_name
# createdb runs inside the container, so the db gets created in the postgres17 container
createdb:
	docker exec -it postgres17 createdb --username=root --owner=root bmc

# deletes the bmc database along with all its data (can't be undone)
dropdb:
	docker exec -it postgres17 dropdb bmc

# -path is the folder with the migration files
# -database is the connection string of the db to run them on
# -verbose prints detailed logs while migrating
# up applies all the pending migrations
migrateup:
	migrate -path db/migrations -database "$(DB_URL)" -verbose up

# down reverts all the applied migrations (rolls the schema back)
migratedown:
	migrate -path db/migrations -database "$(DB_URL)" -verbose down

# reads the settings from sqlc.yaml and generates go code into db/sqlc
# models.go gets a go struct for each table, built from the schema in db/migrations (not the running db)
# each query in db/query becomes a go method; the comment on top tells sqlc its name and what it returns:
#   -- name: CreateAccount :one   -> returns one row
#   :many -> a slice of rows, :exec -> only an error, :execresult -> sql.Result (rows affected etc.)
sqlc:
	sqlc generate

# runs every test in the project (./... means this folder and all folders under it)
# -v (verbose) prints each test's name, PASS/FAIL and anything the test prints
# -cover shows the % of code statements the tests actually ran (e.g. "coverage: 60.0% of statements")
test:
	go test -v -cover ./...

# make treats every target as the name of a file it should build, and skips it if that file already exists
# (e.g. a file named sqlc here would make `make sqlc` print "up to date" and do nothing)
# .PHONY tells make these targets are just commands, not files, so it always runs them
.PHONY: postgres createdb dropdb migrateup migratedown sqlc test
