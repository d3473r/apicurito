# The Angular build output must be present in this directory when building,
# so that it can be embedded into the server binary.
dist:
	rsync -a --delete ../dist/ dist/

all: dist \
     ../target/go/linux-amd64/apicurito-ui \
     ../target/go/darwin-amd64/apicurito-ui \
     ../target/go/windows-amd64/apicurito-ui.exe

../target/go/linux-amd64/apicurito-ui: dist apicurito.go
	GOOS=linux GOARCH=amd64 go build -o $@ .

../target/go/darwin-amd64/apicurito-ui: dist apicurito.go
	GOOS=darwin GOARCH=amd64 go build -o $@ .

../target/go/windows-amd64/apicurito-ui.exe: dist apicurito.go
	GOOS=windows GOARCH=amd64 go build -o $@ .
