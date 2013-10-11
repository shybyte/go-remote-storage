package gors

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"regexp"
	"html/template"
	"strings"
	"strconv"
	"libs/uniuri"
	"io"
	"io/ioutil"
	"crypto/sha512"
	"os"
	"os/user"
	"time"
)

type StorageMode string

const (
	OWNCLOUD = "owncloud"
	HOME     = "home"
)

type Scope struct {
	path  string;
	write bool
}

func (s Scope) String() string {
	if (s.write) {
		return s.path + " (Full Access)"
	}
	return s.path
}

type Authorization struct {
	username      string
	clientId      string
	scopes        []Scope
	bearerToken   string
}

const GORS_PATH = "/gors"

var STORAGE_PATH = GORS_PATH + "/storage/"

var dataPath string
var storageMode StorageMode
var chown string
var resourcesPath string
var externalBaseUrl string

func StartServer(storageDir string, storageModePara StorageMode, chownPara string, resourcesPathPara string, port int, externalBaseUrlPara string) {
	dataPath = storageDir
	storageMode = storageModePara
	chown = chownPara
	resourcesPath = resourcesPathPara
	externalBaseUrl = externalBaseUrlPara
	http.HandleFunc("/.well-known/host-meta.json", handleWebfinger)
	http.HandleFunc(AUTH_PATH, handleAuth)
	http.HandleFunc(STORAGE_PATH, handleStorage)
	http.Handle(GORS_PATH + "/css/", http.StripPrefix(GORS_PATH + "/css/", http.FileServer(http.Dir(resourcesPath + "/css"))))
	err := http.ListenAndServe(":" + strconv.Itoa(port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

/* ------------------------------------ Storage ----------------------------- */

var STORAGE_PATH_PATTERN = regexp.MustCompile("^" + STORAGE_PATH + "([^/]+)(/.*)$")

func handleStorage(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)

	if (r.Method == "OPTIONS") {
		return;
	}

	pathParts := STORAGE_PATH_PATTERN.FindStringSubmatch(r.URL.Path)
	if len(pathParts) < 3 {
		w.WriteHeader(400)
		return;
	}

	username := pathParts[1]
	pathInUserStorage := pathParts[2]

	if !isAuthorized(r, pathInUserStorage) {
		w.WriteHeader(401)
		return;
	}

	userStoragePath := getUserDataPath(username)
	switch r.Method {
	case "GET":
		filename := userStoragePath + pathInUserStorage
		if isDirListingRequest(filename) {
			handleDirectoryListing(w, r, filename)
		} else {
			handleGetFile(w, r, filename)
		}
	case "PUT":
		handlePutFile(w, r, userStoragePath, pathInUserStorage, username)
	case "DELETE":
		handleDeleteFile(w, r, userStoragePath, pathInUserStorage)
	default:
		w.WriteHeader(500)
	}
}

func isDirListingRequest(path string) bool {
	return strings.HasSuffix(path, "/")
}

func isAuthorized(r *http.Request, pathInUserStorage string) bool {
	if r.Method == "GET" && strings.HasPrefix(pathInUserStorage, "/public") && !isDirListingRequest(pathInUserStorage) {
		// everybody can read public data, so we need no authorization
		return true
	} else if getAuthorization(r, pathInUserStorage) != nil {
		return true
	}
	return false
}

func getAuthorization(r *http.Request, pathInUserStorage string) *Authorization {
	// no Bearer Token ?
	if len(r.Header["Authorization"]) == 0 {
		return nil;
	}

	bearerToken := strings.TrimPrefix(r.Header["Authorization"][0], "Bearer ")

	// invalid Bearer Token ?
	authorization := authorizationByBearer[bearerToken]
	if authorization == nil {
		return nil;
	}

	// is Bearer Token valid for user?
	if !strings.HasPrefix(r.URL.Path, STORAGE_PATH + authorization.username) {
		fmt.Println("Token  " + bearerToken + " is invalid for path " + r.URL.Path + "and username " + authorization.username)
		return nil;
	}

	// Is Bearer Token valid for Scopes
	for _, scope := range authorization.scopes {
		if (strings.HasPrefix(pathInUserStorage, "/" + scope.path + "/") ||
				strings.HasPrefix(pathInUserStorage, "/public/" + scope.path + "/") ||
				strings.HasPrefix(scope.path, "root")) &&
				(r.Method == "GET" || (scope.write)) {
			return authorization
		}
	}

	return nil
}

func handleDirectoryListing(w http.ResponseWriter, r *http.Request, dirName string) {
	if needs304Response(r, dirName) {
		w.WriteHeader(304)
		return;
	}

	files, err := ioutil.ReadDir(dirName)

	w.Header().Set("Content-Type", "application/json")

	// Handle non existing and empty dirs
	if err != nil || len(files) == 0 {
		w.WriteHeader(404)
	} else {
		fInfo, _ := os.Stat(dirName)
		addETagFromFileInfo(w, fInfo)
		w.WriteHeader(200)
	}

	fmt.Fprint(w, "{\n")
	realFiles := ignoreMetaFiles(files)
	for i, f := range realFiles {
		fmt.Fprintf(w, `"%s":"%d"`, itemName(f), f.ModTime().Unix())
		if i < len(realFiles) - 1 {
			fmt.Fprintf(w, ",")
		}
		fmt.Fprintf(w, "\n")
	}
	fmt.Fprint(w, "}\n")
}

func ignoreMetaFiles(files []os.FileInfo) []os.FileInfo {
	var realFiles = make([]os.FileInfo,0, len(files)/2)
	for _, f := range files {
		if !strings.HasPrefix(f.Name(), CONTENT_TYPE_FILE_NAME_PREFIX) {
			realFiles = append(realFiles, f)
		}
	}
	return realFiles
}

func handleGetFile(w http.ResponseWriter, r *http.Request, filename string) {
	if needs304Response(r, filename) {
		w.WriteHeader(304)
		return;
	}

	f, err := os.Open(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	contentType, _ := ioutil.ReadFile(contentTypeFilename(filename))
	w.Header().Set("Content-Type", string(contentType))
	fInfo, _ := f.Stat()
	addETagFromFileInfo(w, fInfo)
	http.ServeContent(w, r, fInfo.Name(), fInfo.ModTime(), f)
}

func handlePutFile(w http.ResponseWriter, r *http.Request, userStoragePath string, pathInUserStorage string, username string) {
	filename := userStoragePath + pathInUserStorage;

	if needs412Response(r, filename) {
		w.WriteHeader(412)
		return;
	}

	ensurePath(filename, username)
	f, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error", err)
		w.WriteHeader(500)
		return
	}
	defer f.Close()
	io.Copy(f, r.Body)
	err = ioutil.WriteFile(contentTypeFilename(filename), []byte(r.Header.Get("Content-Type")), 0644)
	chownIfNeeded(contentTypeFilename(filename), username);
	markAncestorFoldersAsModified(userStoragePath, pathInUserStorage)
	chownAncestorFoldersIfNeeded(userStoragePath, pathInUserStorage, username)
	addETag(w, filename)
	chownIfNeeded(filename, username);
	w.WriteHeader(200)
}

func chownIfNeeded(filename string, username string) {
	if chown == "" {
		return;
	} else if (chown != "@") {
		username = chown
	}
	user, err := user.Lookup(username)
	if err != nil {
		fmt.Println("Error while chown. Can't find user:", err)
	}
	uid, _ := strconv.Atoi(user.Uid)
	gid, _ := strconv.Atoi(user.Gid)
	err = os.Chown(filename, uid, gid)
	if err != nil {
		fmt.Println("Error while chown:", err, user)
	}
}



func chownAncestorFoldersIfNeeded(basePath, modifiedPath string, username string) {
	forAllAncestorFolders(basePath, modifiedPath, func(path string) {
			chownIfNeeded(path, username)
		})
}

func needs304Response(r *http.Request, filename string) bool {
	if ifNoneMatch := r.Header.Get("If-None-Match"); len(ifNoneMatch) > 0 {
		fInfo, err := os.Stat(filename)
		if (err == nil && getETag(fInfo) == ifNoneMatch) {
			return true
		}
	}
	return false
}

func needs412Response(r *http.Request, filename string) bool {
	if ifMatch := r.Header.Get("If-Match"); len(ifMatch) > 0 {
		fInfo, err := os.Stat(filename)
		if (err != nil || getETag(fInfo) != ifMatch) {
			return true
		}
	}
	if ifNoneMatch := r.Header.Get("If-None-Match"); ifNoneMatch == "*" {
		_, err := os.Stat(filename)
		if (err == nil) {
			return true
		}
	}
	return false
}

func handleDeleteFile(w http.ResponseWriter, r *http.Request, userStoragePath string, pathInUserStorage string) {
	filename := userStoragePath + pathInUserStorage;
	if needs412Response(r, filename) {
		w.WriteHeader(412)
		return;
	}
	existFile, err := exists(filename)
	if (!existFile) {
		w.WriteHeader(404)
		return;
	} else if (err != nil) {
		w.WriteHeader(500)
		return;
	}
	addETag(w, filename)
	os.Remove(filename)
	os.Remove(contentTypeFilename(filename))
	markAncestorFoldersAsModified(userStoragePath, pathInUserStorage)
	removeEmptyAncestorFolders(userStoragePath, pathInUserStorage)
}

func addETag(w http.ResponseWriter, filename string) {
	fInfo, _ := os.Stat(filename)
	addETagFromFileInfo(w, fInfo)
}

func addETagFromFileInfo(w http.ResponseWriter, fInfo os.FileInfo) {
	w.Header().Set("ETag", getETag(fInfo))
}

func getETag(fInfo os.FileInfo) string {
	return fmt.Sprintf("\"%d\"", fInfo.ModTime().Unix())
}

var FILE_NAME_PATTERN = regexp.MustCompile("/([^/]+)$")

const CONTENT_TYPE_FILE_NAME_PREFIX = ".rsct."

func contentTypeFilename(filename string) string {
	return FILE_NAME_PATTERN.ReplaceAllString(filename, "/" + CONTENT_TYPE_FILE_NAME_PREFIX + "$1")    //rsct = RemoteStorageContentType
}

func markAncestorFoldersAsModified(basePath, modifiedPath string) {
	time := time.Now()
	forAllAncestorFolders(basePath, modifiedPath, func(path string) {
			os.Chtimes(path, time, time)
		})
}

func forAllAncestorFolders(basePath, modifiedPath string, f func (string)) {
	modifiedPathParts := strings.Split(modifiedPath[1:], "/")
	currentPath := basePath;
	for _, pathPart := range modifiedPathParts[:len(modifiedPathParts) - 1] {
		currentPath = currentPath + "/" + pathPart
		f(currentPath)
	}
}

var LAST_PATH_PART_PATTERN = regexp.MustCompile("(/[^/]+)$")

func removeEmptyAncestorFolders(basePath, path string) {
	currentPath := LAST_PATH_PART_PATTERN.ReplaceAllString(basePath + path, "")
	for ; len(currentPath) > len(basePath); currentPath = LAST_PATH_PART_PATTERN.ReplaceAllString(currentPath, "") {
		files, _ := ioutil.ReadDir(currentPath)
		if (len(files) == 0) {
			os.Remove(currentPath)
		}
	}
}

func ensurePath(filename string, username string) {
	path := filename[:strings.LastIndex(filename, "/")]
	os.MkdirAll(path, os.ModePerm)
	chownIfNeeded(path, username)
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
	return false, err
}

func itemName(f os.FileInfo) string {
	if f.IsDir() {
		return f.Name() + "/"
	}
	return f.Name()
}

func getUserDataPath(username string) string {
	return userGorsDir(username) + "data"
}

func userGorsDir(username string) string {
	if storageMode == OWNCLOUD {
		return dataPath + "/" + username + "/files/.gors/"
	}
	return dataPath + "/" + username + "/.gors/"
}

/* ------------------------------------ Auth ----------------------------- */

var authorizationByBearer = make(map[string]*Authorization)
var AUTH_PATH = GORS_PATH + "/auth/"

func handleAuth(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Path[len(AUTH_PATH):]
	query := r.URL.Query()
	scopes := parseScopes(query["scope"][0])
	wrongPassword := false

	if (r.Method == "POST") {
		r.ParseForm()
		if (isPasswordValid(username, r.Form["password"][0])) {
			authorization := Authorization{username, query["client_id"][0], scopes, uniuri.NewLen(10)}
			authorizationByBearer[authorization.bearerToken] = &authorization
			http.Redirect(w, r , query["redirect_uri"][0] + "#access_token=" + authorization.bearerToken, 301)
			return
		} else {
			wrongPassword = true
		}
	}

	t, _ := template.ParseFiles(resourcesPath + "/templates/login.html")
	t.Execute(w, map[string]interface{} {
			"username": username,
			"scopes": scopes,
			"clientID": query["client_id"][0],
			"wrongPassword": wrongPassword,
		})
}

func isPasswordValid(username string, password string) bool {
	passwordFileBuf, _ := ioutil.ReadFile(userGorsDir(username) + "password-sha512.txt")
	expectedPasswordSha1 := strings.Trim(string(passwordFileBuf), " \n")
	return expectedPasswordSha1 == sha512Sum(password)
}

func sha512Sum(s string) string {
	sha512Hash := sha512.New()
	io.WriteString(sha512Hash, s)
	return fmt.Sprintf("%x", sha512Hash.Sum(nil))
}


func parseScopes(scopesString string) []Scope {
	scopeStrings := strings.Split(scopesString, " ")
	scopes := make([]Scope, len(scopeStrings))
	for i, scopeString := range scopeStrings {
		parts := strings.Split(scopeString, ":")
		if (parts[1] == "rw") {
			scopes[i] = Scope{parts[0], true}
		} else {
			scopes[i] = Scope{parts[0], false}
		}
	}
	return scopes
}

/* ------------------------------------ Webfinger ------------------------ */


var RESOURCE_PARA_PATTERN = regexp.MustCompile(`^acct:(.+)@(.+)$`)

func handleWebfinger(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)
	w.Header().Set("Content-Type", "application/json")
	fmt.Println(r)
	username := RESOURCE_PARA_PATTERN.FindStringSubmatch(r.URL.Query()["resource"][0])[1]
	fmt.Fprintf(w, createWebfingerJson(getBaseUrl(r), username))
}

func getBaseUrl(r *http.Request) string {
	if externalBaseUrl != "" {
		return externalBaseUrl
	}
	return getUsedProtocol(r) + "://" + getOwnHost(r)
}

func getUsedProtocol(r *http.Request) string {
	if strings.HasSuffix(getOwnHost(r), ":443") {
		return "https"
	}
	return "http"
}

func getOwnHost(r *http.Request) string {
	if len(r.Header["X-Forwarded-Host"]) > 0 {
		return r.Header["X-Forwarded-Host"][0]
	}
	return r.Host
}

func createWebfingerJson(baseURL, username string) string {
	b, _ := json.Marshal(map[string]interface{}{
		"links": []interface{}{
			map[string]interface{} {
				"href": baseURL + STORAGE_PATH + username,
				"rel": "remoteStorage",
				"type":"https://www.w3.org/community/rww/wiki/read-write-web-00#simple",
				"properties": map[string]string{
					"auth-method": "https://tools.ietf.org/html/draft-ietf-oauth-v2-26#section-4.2",
					"auth-endpoint":  baseURL + AUTH_PATH + username,
				},
			},
		},
	})
	return string(b)
}

/* ------------------------------------ CORS ------------------------ */

func enableCORS(w http.ResponseWriter, r *http.Request) {
	var origin string
	if len(r.Header["Origin"]) > 0 {
		origin = r.Header["Origin"][0]
	} else {
		origin = "*"
	}
	//fmt.Println(r);
	//fmt.Println("Origin:" + origin);
	header := w.Header()
	header.Add("access-control-allow-origin", origin)
	header.Add("access-control-allow-headers", "content-type, authorization, origin")
	header.Add("access-control-allow-methods", "GET, PUT, DELETE")
}
