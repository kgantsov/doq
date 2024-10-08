build_web:
	cd admin_ui; npm run build; cp -r dist/* ../cmd/server

build_linux_amd64:
	cd admin_ui; npm run build; cp -r dist/* ../cmd/server
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/doq cmd/server/main.go

build_darwin_arm64:
	cd admin_ui; npm run build; cp -r dist/* ../cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/doq cmd/server/main.go

build_dev:
	cd admin_ui; npm run build; cp -r dist/* ../cmd/server
	go build -o cmd/server/doq cmd/server/main.go

proto_compile:
	protoc pkg/proto/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.
