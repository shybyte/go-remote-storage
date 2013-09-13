#setcap cap_net_bind_service=ep /path/to/the/executable
build:
	export GOPATH=`pwd`
	cd src/main; go install
init-storage:
	rm -rf tmp/storage
	mkdir tmp/storage
	cp -r storage-example/* tmp/storage/
