# Remote Storage Implementation written in golang
## Installation
TODO
### Creating Password
echo -n password | sha512sum | cut -f 1 -d " " >password-sha512.txt
### Example Apache Proxy Config
ProxyPass /.well-known/host-meta.json http://localhost:8888/.well-known/host-meta.json
ProxyPassReverse /.well-known/host-meta.json http://localhost:8888/.well-known/host-meta.json
ProxyPass /auth http://localhost:8888/auth
ProxyPassReverse /auth http://localhost:8888/auth
ProxyPass /storage http://localhost:8888/storage
ProxyPassReverse /storage http://localhost:8888/storage
ProxyPass /css http://localhost:8888/css
ProxyPassReverse /css http://localhost:8888/css