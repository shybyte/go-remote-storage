#setcap cap_net_bind_service=ep /path/to/the/executable
build:
	cd src/main;GOPATH=`pwd`/../.. go install

clean-storage:
	rm -rf tmp/storage
	mkdir tmp/storage

init-storage-home: clean-storage
	cp -r storage-examples/home/* tmp/storage/

init-storage-owncloud: clean-storage
	cp -r storage-examples/owncloud-data/* tmp/storage/

run-owncloud: build
	sudo ./bin/main -storage /var/www/owncloud/data -mode owncloud -port 8888 -chown www-data -resources src

run-home: build
	sudo ./bin/main -storage tmp/storage  -mode home -port 8888 -chown @ -resources src
