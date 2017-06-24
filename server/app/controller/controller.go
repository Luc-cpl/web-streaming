package controller

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

//ExemploGet é um exemplo do como ler dados de requisições GET
func ExemploGet(w http.ResponseWriter, r *http.Request) {
	x := mux.Vars(r)
	fmt.Println(x)
}
