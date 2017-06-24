package main

import (
	"fmt"
	"net/http"
	"time"

	mgoS "github.com/Luc-cpl/mgoSimpleCRUD"
	"github.com/Luc-cpl/web-streaming/server/app"
	"github.com/Luc-cpl/web-streaming/server/app/controller"
	mgo "gopkg.in/mgo.v2"
)

var serverPort = ":8080"

func main() {

	//Database connection info (eddit if necessary)
	maxWait := time.Duration(5 * time.Second)
	local, err := mgo.DialWithTimeout("localhost:27017", maxWait)
	if err == nil {
		fmt.Println("Connected to MongoDB.")
		defer local.Close()
		mgoS.DB.Session = local.Clone()
		mgoS.DB.Database = "teste"
		mgoS.DB.UserIdentityValue = "email"
		mgoS.DB.AuthMap, err = mgoS.LoadAuthMap("./authMap.json", true)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		fmt.Println("Unable to connect to local mongo instance!")
	}

	controller.Streaming = controller.NewHub()

	router := routes.NewRouter()

	fmt.Println("started server at port", serverPort)
	http.ListenAndServe(serverPort, router)
}
