package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/Luc-cpl/web-streaming/server/render"

	"github.com/gorilla/mux"
)

//The URL to redirect if the JSON redirect is true
var redirectURL = "/"

type viewMap struct {
	URL      string            `json:"url"`      //the access url
	Template string            `json:"template"` //the html template to pass in view folder
	Auth     bool              `json:"auth"`     //if login is necessary
	Redirect bool              `json:"redirect"` //to redirect if is logged
	Files    map[string]string `json:"files"`    //files to render in template
}

//Views controll all the url views in webapp
func Views(w http.ResponseWriter, r *http.Request) {
	url := mux.Vars(r)["rest"]

	var data interface{}
	if strings.Contains(url, "/request:") {
		arr := strings.SplitN(url, "/request:", 2)
		url = arr[0]
		data = getRequest(arr[1], GetUser(r))
	} else if strings.HasPrefix(url, "request:") {
		data = getRequest(strings.Trim(url, "request:"), GetUser(r))
		url = ""
	}

	raw, err := ioutil.ReadFile("./views/viewmap.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var decodedJSON []viewMap

	err = json.Unmarshal(raw, &decodedJSON)
	if err != nil {
		fmt.Println(err.Error())
	}

	user := GetUser(r)

	view, auth, exist := findFiles(decodedJSON, url, user.ID, w, r)

	if auth == true && exist == true {
		render.Render(w, data, view.Template, view.Files)
		return
	} else if exist == true {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	} else {
		if strings.ContainsAny(url, ".") {
			http.ServeFile(w, r, "./public/"+url)
			return
		}
		view, _, _ = findFiles(decodedJSON, "404", "", w, r)
		render.Render(w, nil, view.Template, view.Files)
		return

	}

}

func findFiles(decodedJSON []viewMap, url string, userID string, w http.ResponseWriter, r *http.Request) (view viewMap, auth bool, exist bool) {
	for _, element := range decodedJSON {
		if strings.EqualFold(element.URL, url) {
			exist = true
			if element.Auth == false {
				if element.Redirect == true && !strings.EqualFold(userID, "") {
					if ("/" + url) != redirectURL {
						http.Redirect(w, r, redirectURL, http.StatusSeeOther)
						return view, false, false
					}
					auth = false
				} else {
					auth = true
					return element, auth, exist
				}

			} else if !strings.EqualFold(userID, "") {
				auth = true
				return element, auth, exist
			}

		}
	}
	return view, false, false
}
