package main

import (
	"io/ioutil"
	"net/http"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

func init() {
	http.HandleFunc("/authorization", authorizeHandler)
	http.HandleFunc("/token", tokenHandler)
}

func authorizeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	log.Warningf(ctx, r.URL.RawQuery)
	url := "https://identint.familysearch.org/cis-web/oauth2/v3/authorization" + "?" + r.URL.RawQuery
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Remove client_secret from form data. Google Assistant always sends it, but FS doesn't want it.
func tokenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	r.ParseForm()
	r.Form.Del("client_secret")
	body := r.Form.Encode()

	newReq, err := http.NewRequest("POST", "https://identint.familysearch.org/cis-web/oauth2/v3/token", strings.NewReader(body))
	newReq.Header.Set("Accept", "application/json")

	client := urlfetch.Client(ctx)
	resp, err := client.Do(newReq)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	recievedBody, err := ioutil.ReadAll(resp.Body)
	log.Warningf(ctx, "Received body: %s", string(recievedBody))
	if err != nil {
		http.Error(w, "Bad response from FamilySearch", http.StatusInternalServerError)
		return
	}

	w.Write(recievedBody)
}
