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
	"crypto/sha1"
	"os"
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

var dataPath string

func StartServer(storageDir string, port int) {
	fmt.Println(storageDir)
	dataPath = storageDir
	http.HandleFunc("/.well-known/host-meta.json", handleWebfinger)
	http.HandleFunc("/auth/", handleAuth)
	http.HandleFunc("/storage/", handleStorage)
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("src/css"))))
	err := http.ListenAndServe(":" + strconv.Itoa(port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

/* ------------------------------------ Storage ----------------------------- */

func handleStorage(w http.ResponseWriter, r *http.Request) {
	enableCORS(w, r)

	if (r.Method == "OPTIONS") {
		return;
	}

	// no Bearer Token ?
	if len(r.Header["Bearer"]) == 0 {
		w.WriteHeader(401)
		return;
	}

	// invalid Bearer Token ?
	authorization := authorizationByBearer[r.Header["Bearer"][0]]
	if authorization == nil {
		w.WriteHeader(401)
		return;
	}
	pathPrefix := "/storage/" + authorization.username + "/"
	if !strings.HasPrefix(r.URL.Path, pathPrefix) {
		w.WriteHeader(401)
		return;
	}

	files, err := ioutil.ReadDir(getUserDataPath(authorization.username)+ r.URL.Path[len(pathPrefix)-1:])
	w.WriteHeader(200)
	fmt.Println("Files:")
	fmt.Println(err)
	fmt.Println(files)
	fmt.Fprint(w, "{\n")
	for i, f := range files {
		fmt.Println(f.Name(), f.IsDir(), f.ModTime(), f.ModTime().Unix())
		fmt.Fprintf(w, `"%s":"%d"`, itemName(f), f.ModTime().Unix())
		if i<len(files)-1 {
			fmt.Fprintf(w,",")
		}
		fmt.Fprintf(w,"\n")
	}
	fmt.Fprint(w, "}\n")

}

func itemName(f os.FileInfo)  string{
	if f.IsDir() {
		return f.Name()+"/"
	}
	return f.Name()
}

func getUserDataPath(username string) string {
	return userGorsDir(username) + "data"
}

func userGorsDir(username string) string {
	return dataPath + "/" + username + "/.gors/"
}

/* ------------------------------------ Auth ----------------------------- */

var authorizationByBearer = make(map[string]*Authorization)

func handleAuth(w http.ResponseWriter, r *http.Request) {
	fmt.Println(authorizationByBearer)
	username := r.URL.Path[len("/auth/"):]
	query := r.URL.Query()
	scopes := parseScopes(query["scope"][0])
	wrongPassword := false

	if (r.Method == "POST") {
		r.ParseForm()
		fmt.Println(r.Form)
		if (isPasswordValid(username, r.Form["password"][0])) {
			authorization := Authorization{username, query["client_id"][0], scopes, uniuri.NewLen(10)}
			authorizationByBearer[authorization.bearerToken] = &authorization
			http.Redirect(w, r , query["redirect_uri"][0] + "#access_token=" + authorization.bearerToken, 301)
			return
		} else {
			wrongPassword = true
		}
	}

	t, _ := template.ParseFiles("src/templates/login.html")
	t.Execute(w, map[string]interface{} {
			"username": username,
			"scopes": scopes,
			"clientID": query["client_id"][0],
			"wrongPassword": wrongPassword,
		})
}

func isPasswordValid(username string, password string) bool {
	passwordFileBuf, _ := ioutil.ReadFile(userGorsDir(username) + "password-sha1.txt")
	expectedPasswordSha1 := string(passwordFileBuf)
	return expectedPasswordSha1[:40] == sha1Sum(password)
}

func sha1Sum(s string) string {
	sha1Hash := sha1.New()
	io.WriteString(sha1Hash, s)
	return fmt.Sprintf("%x", sha1Hash.Sum(nil))
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
	fmt.Fprintf(w, createWebfingerJson(getOwnHost(r), username))
}

func getOwnHost(r *http.Request) string {
	if len(r.Header["X-Forwarded-Host"]) > 0 {
		return r.Header["X-Forwarded-Host"][0]
	}
	return r.Host
}

func createWebfingerJson(host, username string) string {
	baseURL := "http://" + host
	b, _ := json.Marshal(map[string]interface{}{
		"links": []interface{}{
			map[string]interface{} {
				"href": baseURL + "/storage/" + username,
				"rel": "remoteStorage",
				"type":"https://www.w3.org/community/rww/wiki/read-write-web-00#simple",
				"properties": map[string]string{
					"auth-method": "https://tools.ietf.org/html/draft-ietf-oauth-v2-26#section-4.2",
					"auth-endpoint":  baseURL + "/auth/" + username,
				},
			},
		},
	})
	return string(b)
}

/* ------------------------------------ CORS ------------------------ */

func enableCORS(w http.ResponseWriter, r *http.Request) {
	var origin string
	if len(r.Header["origin"]) > 0 {
		origin = r.Header["origin"][0]
	} else {
		origin = "*"
	}
	header := w.Header()
	header.Add("access-control-allow-origin", origin)
	header.Add("access-control-allow-headers", "content-type, authorization, origin")
	header.Add("access-control-allow-methods", "GET, PUT, DELETE")
}
