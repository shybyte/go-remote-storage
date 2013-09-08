import re
import urllib
import urllib2
import httplib
import json

server = "http://localhost:8888"

def test_get_webfinger():
	request = urllib2.urlopen(server+"/.well-known/host-meta.json?resource=acct%3Ausername%40domain.net");
	response = request.read();
	print response
	print request.info()
	resultJSON = json.loads(response)
	link = resultJSON['links'][0]
	assert link["href"] == server + "/storage/username"
	assert link["rel"] == "remoteStorage"
	assert link["type"] == "https://www.w3.org/community/rww/wiki/read-write-web-00#simple"
	props = link["properties"]
	assert props["auth-method"] == "https://tools.ietf.org/html/draft-ietf-oauth-v2-26#section-4.2"
	assert props["auth-endpoint"] == server + "/auth/username"
	assert request.info()['Content-Type'].startswith('application/json')
	

def test_auth_page():
	request = urllib2.urlopen(server+"/auth/marco"+"?redirect_uri=https%3A%2F%2Fmyfavoritedrinks.5apps.com%2F&client_id=myfavoritedrinks.5apps.com&scope=myfavoritedrinks%3Arw&response_type=token");
	response = request.read();
	assert "<h1>Allow Remote Storage Access?</h1>" in response
	assert "marco" in response
	assert "myfavoritedrinks" in response
	assert "Full Access" in response
	assert "myfavoritedrinks.5apps.com" in response

def test_confirm_permission_with_password_and_redirect_to_app():
	values = {'password' : 'password'}
	data = urllib.urlencode(values)
	headers = {"Content-type": "application/x-www-form-urlencoded",
            "Accept": "text/plain"}	
	conn = httplib.HTTPConnection('localhost:8888')
	conn.request("POST", "/auth/user1"+"?redirect_uri=https%3A%2F%2Fmyfavoritedrinks.5apps.com%2F&client_id=myfavoritedrinks.5apps.com&scope=myfavoritedrinks%3Arw&response_type=token",data,headers)
	r = conn.getresponse()
	print r.status, r.reason
	redirectUrl = r.getheader('Location')
	expectedRedirectUrlPrefix = 'https://myfavoritedrinks.5apps.com/#access_token='
	assert redirectUrl.startswith(expectedRedirectUrlPrefix)
	assert len(redirectUrl[len(expectedRedirectUrlPrefix):])>=10

	
