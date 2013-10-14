ab -n 20 -c 10 -H "Authorization: Bearer 2qcvnf4zKY" "http://auerpi.no-ip.biz:8888/gors/storage/shybyte/myfavoritedrinks/green-tea" >benchmark-results/gors-http.txt
ab -n 20 -c 10 -H "Authorization: Bearer 2qcvnf4zKY" "https://auerpi.no-ip.biz:443/gors/storage/shybyte/myfavoritedrinks/green-tea" >benchmark-results/gors-https-apache-proxy.txt
ab -k -n 20 -c 10 -H "Authorization: Bearer eYEoSAtse0" "https://auerpi.no-ip.biz:443/gors/storage/shybyte/myfavoritedrinks/green-tea" >benchmark-results/gors-https-apache-proxy-keep-alive.txt
ab -k -n 20 -c 1 -H "Authorization: Bearer eYEoSAtse0" "https://auerpi.no-ip.biz:443/gors/storage/shybyte/myfavoritedrinks/green-tea" >benchmark-results/gors-https-apache-proxy-keep-alive-c1.txt
