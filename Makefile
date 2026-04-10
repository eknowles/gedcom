test:
 	# Run all the tests with the race detector enabled.
	go test -race ./...

test-unit:
	# Run only unit tests (skip integration tests that require external services).
	go test -short -race ./...

test-coverage:
	echo "" > coverage.txt

	for d in $$(go list ./... | grep -v vendor); do \
		go test -short -race -coverprofile=profile.out -covermode=atomic $$d || exit 1; \
		if [ -f profile.out ]; then \
			cat profile.out >> coverage.txt; \
			rm profile.out; \
		fi \
	done

zip:
	sed -i '' "s/unknown version/$(TAG)/" cmd/gedcom/version.go
	rm -rf bin
	mkdir bin
	go build -o bin/gedcom$(EXT) ./cmd/gedcom
	zip gedcom-$(NAME).zip -r bin

sql:
	go build -o ./bin/gedcom ./cmd/gedcom
	./bin/gedcom surrealdb -gedcom examples/gedcom.ged -output examples/gedcom.surql -allow-invalid-indents
	#./bin/gedcom publish -gedcom ./gedcom.ged -output-dir output -allow-invalid-indents
	echo 'REMOVE DATABASE IF EXISTS main;' | surreal sql --hide-welcome --pretty -u root -p root --ns main --db main
	surreal import -e http://localhost:8000/sql -u root -p root --ns main --db main \
		examples/gedcom.surql

.PHONY: test test-unit zip
