#setcap cap_net_bind_service=ep /path/to/the/executable
build:
	cd src/main;GOPATH=`pwd`/../.. go install
init-storage:
	rm -rf tmp/storage
	mkdir tmp/storage
	#cp -r storage-examples/home/* tmp/storage/
	cp -r storage-examples/owncloud-data/* tmp/storage/
run-owncloud: build
	bin/main -storage /var/www/owncloud/data -mode owncloud -port 9000 -chown
