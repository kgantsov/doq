test:
	go test ./... -race -cover

bench:
	go test ./... -bench=. -benchmem

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

run_web:
	cd admin_ui; npm run dev

run_node_0:
	cd cmd/server ; go run . --storage.data_dir data --cluster.node_id node-0 --http.port 8000 --raft.address 127.0.0.1:9000 --grpc.address 127.0.0.1:10000

run_node_1:
	cd cmd/server ; go run . --storage.data_dir data --cluster.node_id node-1 --http.port 8001 --raft.address 127.0.0.1:9001 --grpc.address 127.0.0.1:10001 --cluster.join_addr 127.0.0.1:8000

run_node_2:
	cd cmd/server ; go run . --storage.data_dir data --cluster.node_id node-2 --http.port 8002 --raft.address 127.0.0.1:9002 --grpc.address 127.0.0.1:10002 --cluster.join_addr 127.0.0.1:8000
