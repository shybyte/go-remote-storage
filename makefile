#setcap cap_net_bind_service=ep /path/to/the/executable
build:
	export GOPATH=`pwd`
	cd src/main; go install