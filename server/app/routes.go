package routes

import (
	"net/http"

	"github.com/Luc-cpl/web-streaming/server/app/controller"

	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {

	var routes = Routes{
		//API
		Route{"UserLogin", "POST", "/api/users/login", controller.UserLogin},
		Route{"NewUser", "POST", "/api/users/new", controller.NewUser},
		Route{"ChangeIdentityValue", "POST", "/api/users/changeIdentityValue", controller.ChangeIdentityValue},
		Route{"ChangePassword", "POST", "/api/users/ChangePassword", controller.ChangePassword},

		//Websocket
		Route{"NewWebsocket", "GET", "/ws", controller.NewWebsocket},
		Route{"Clientbsocket", "GET", "/ws/client/{rest:.*}", controller.StartClientWebsocket},
		Route{"UserWebsocket", "GET", "/ws/user/{rest:.*}", controller.StartUserWebsocket},

		//views manager
		Route{"Index", "GET", "/", controller.Views},
		Route{"Views", "GET", "/{rest:.*}", controller.Views},
	}

	router := mux.NewRouter().StrictSlash(true)

	for _, route := range routes {
		router.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(route.HandlerFunc)
	}

	return router

}

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route
