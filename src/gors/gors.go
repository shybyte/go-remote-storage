package gors

import (
	"fmt"
	"net/http"
	"encoding/json"
	"regexp"
	"html/template"
	"strings"
	"libs/uniuri"
)

type Scope struct {
	path  string;
	write bool
}

func (s Scope) String() string {
	if (s.write) {
		return s.path+" (Full Access)"
	} else {
		return s.path;
	}
}

type Authorization struct {
	username string
	clientId string
	scopes   []Scope
}

var authorizationByBearer = make(map[string]Authorization)

func StartServer() {
	http.HandleFunc("/.well-known/host-meta.json", handleWebfinger)
	http.HandleFunc("/auth/", handleAuth)
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("src/css"))))
	http.ListenAndServe(":8888", nil)
}

/* ------------------ Auth ----------------------------- */
func handleAuth(w http.ResponseWriter, r *http.Request) {
	fmt.Println(authorizationByBearer)
	username := r.URL.Path[len("/auth/"):]
	query := r.URL.Query()
	scopes := parseScopes(query["scope"][0])

	if (r.Method == "POST") {
		r.ParseForm()
		fmt.Println(r.Form)
		scopes2 := []Scope{}
		authorizationByBearer[uniuri.NewLen(10)] = Authorization{username, query["client_id"][0], scopes2}
		http.Redirect(w, r , "http://blog.fefe.de", 301)
		return
	}

	t, _ := template.ParseFiles("src/templates/login.html")
	t.Execute(w, map[string]interface{} {
			"username": username,
			"scopes": scopes,
			"clientID": query["client_id"][0],
		})
}

func parseScopes(scopesString string) []Scope {
	scopeStrings := strings.Split(scopesString," ")
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

/* ------------------ Webfinger ------------------------ */

var RESOURCE_PARA_PATTERN = regexp.MustCompile(`^acct:(.+)@(.+)$`)

func handleWebfinger(w http.ResponseWriter, r *http.Request) {
	username := RESOURCE_PARA_PATTERN.FindStringSubmatch(r.URL.Query()["resource"][0])[1]
	fmt.Fprintf(w, createWebfingerJson(r.Host, username))
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
