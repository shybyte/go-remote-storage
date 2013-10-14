# Remote Storage Implementation written in golang
## Installation
TODO

### Creating Password
echo -n password | sha512sum | cut -f 1 -d " " >password-sha512.txt

### Apache Reverse Proxy Config
a2enmod proxy
a2enmod proxy_http

#### Example Apache Proxy Config
ProxyPass /.well-known/host-meta.json http://localhost:8888/.well-known/host-meta.json
ProxyPassReverse /.well-known/host-meta.json http://localhost:8888/.well-known/host-meta.json
ProxyPass /gors http://localhost:8888/gors
ProxyPassReverse /gors http://localhost:8888/gors

service apache2 restart