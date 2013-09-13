import re
import urllib
import urllib2
import httplib
import json
import shutil
import os
import pytest

port = "8889"
server = "http://localhost:"+port



@pytest.fixture
def givenTestStorage():
    copy_and_overwrite("../storage-example","../tmp/test-storage")


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
	conn = httplib.HTTPConnection('localhost:'+port)
	conn.request("POST", "/auth/user1"+"?redirect_uri=https%3A%2F%2Fmyfavoritedrinks.5apps.com%2F&client_id=myfavoritedrinks.5apps.com&scope=myfavoritedrinks%3Arw&response_type=token",data,headers)
	r = conn.getresponse()
	print r.status, r.reason
	redirectUrl = r.getheader('Location')
	expectedRedirectUrlPrefix = 'https://myfavoritedrinks.5apps.com/#access_token='
	assert redirectUrl.startswith(expectedRedirectUrlPrefix)
	assert len(redirectUrl[len(expectedRedirectUrlPrefix):])>=10

def test_storage_cors():
	conn = httplib.HTTPConnection('localhost:'+port)
	conn.request("OPTIONS", "/storage/user1/myfavoritedrinks/")
	r = conn.getresponse()
	assert r.status == 200;
	assert r.getheader('Access-Control-Allow-Origin') == "*"

def test_storage_cors():
	r = makeRequest("/storage/user1/myfavoritedrinks/",'OPTIONS')
	assert r.status == 200;
	assert r.getheader('Access-Control-Allow-Origin') == "*"

def test_storage_directory_listing_needs_bearer_token(givenTestStorage):	
	r = makeRequest("/storage/user1/myfavoritedrinks/")
	assert r.status == 401;

def test_storage_directory_listing_needs_valid_bearer_token(givenTestStorage):
	r = makeRequest("/storage/user1/myfavoritedrinks/",'GET',{'Bearer': "invalid"})
	assert r.status == 401;
	
def test_storage_directory_listing_needs_bearer_token_matching_user(givenTestStorage):
	bearerToken = requestBearerToken()
	r = makeRequest("/storage/otheruser/myfavoritedrinks/",'GET',{'Bearer': bearerToken})
	assert r.status == 401;	

def test_storage_directory_listing(givenTestStorage):
	bearerToken = requestBearerToken()
	r = makeRequest("/storage/user1/module/",'GET',{'Bearer': bearerToken})	
	assert r.status == 200;
	dirList = json.loads(r.read())
	assert dirList['file.txt']
	assert dirList['dir/']

def test_storage_directory_listing_for_non_existing_fir(givenTestStorage):
	bearerToken = requestBearerToken()
	r = makeRequest("/storage/user1/notextisting/",'GET',{'Bearer': bearerToken})	
	assert r.status == 200;
	dirList = json.loads(r.read())
	assert len(dirList) == 0


# utils
def requestBearerToken():
	values = {'password' : 'password'}
	data = urllib.urlencode(values)
	headers = {"Content-type": "application/x-www-form-urlencoded"}	
	conn = httplib.HTTPConnection('localhost:'+port)
	conn.request("POST", "/auth/user1"+"?redirect_uri=https%3A%2F%2Fmyfavoritedrinks.5apps.com%2F&client_id=myfavoritedrinks.5apps.com&scope=myfavoritedrinks%3Arw&response_type=token",data,headers)
	r = conn.getresponse()		
	redirectUrl = r.getheader('Location')
	expectedRedirectUrlPrefix = 'https://myfavoritedrinks.5apps.com/#access_token='
	return redirectUrl[len(expectedRedirectUrlPrefix):]


def makeRequest(path,method="GET",headers={}):
	conn = httplib.HTTPConnection('localhost:'+port)
	conn.request(method, path,"",headers)
	return conn.getresponse()
	
def copy_and_overwrite(from_path, to_path):
    if os.path.exists(to_path):
        shutil.rmtree(to_path)
    shutil.copytree(from_path, to_path)
