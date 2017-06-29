package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"strings"

	mgoS "github.com/Luc-cpl/mgoSimpleCRUD"
	"github.com/gorilla/securecookie"
)

//O Json "logged" indica que o cliente fez login, se falhar fazer verificação pela existência do cookie
//A função do model userData "GetUserID"

//Cookie é uma variavel para utilização e criação de cookies segundo uma chae aleatória criada
var Cookie = securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32))

type Response struct {
	Login bool  `json:"login"`
	Err   error `json:"err"`
}

var response Response

//UserLogin faz o login do usuário depois de passar as informações "email" e "password" por uma requisição POST
func UserLogin(w http.ResponseWriter, r *http.Request) {
	var user mgoS.User
	user, err := mgoS.DB.LoginUser(r.PostFormValue(mgoS.DB.UserIdentityValue), r.PostFormValue("password"))
	if err != nil {
		response.Login = false
		response.Err = err
		json.NewEncoder(w).Encode(response)
		return
	} else if encoded, err := Cookie.Encode("session", user); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
			//MaxAge: 1800, //duração do cookie
		}
		http.SetCookie(w, cookie)
		response.Login = true
		json.NewEncoder(w).Encode(response)
	}
}

//NewUser cria um novo usuário no banco de dados
func NewUser(w http.ResponseWriter, r *http.Request) {
	var user mgoS.User
	var err error
	user.ID, err = mgoS.DB.CreateUser(r.PostFormValue(mgoS.DB.UserIdentityValue), r.PostFormValue("password"))
	if err != nil {
		response.Login = false
		response.Err = err
		json.NewEncoder(w).Encode(response)
		return
	} else if encoded, err := Cookie.Encode("session", user); err == nil {
		cookie := &http.Cookie{
			Name:   "session",
			Value:  encoded,
			Path:   "/",
			MaxAge: 1800, //duração do cookie
		}
		http.SetCookie(w, cookie)
		response.Login = true
		json.NewEncoder(w).Encode(response)
	}

}

//ChangeIdentityValue receive a new identity value, checks if exist and changes the old value to the new one
func ChangeIdentityValue(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	newValue := r.PostFormValue(mgoS.DB.UserIdentityValue)

	checkByt := []byte(`{"` + mgoS.DB.UserIdentityValue + `": "` + newValue + `"}`)
	var checkInterface interface{}
	err := json.Unmarshal(checkByt, &checkInterface)

	newSession := mgoS.DB.Session.Copy()
	defer newSession.Close()
	n, err := newSession.DB(mgoS.DB.Database).C("users").Find(checkInterface).Count()

	if err != nil {
		json.NewEncoder(w).Encode(err)
		return
	}

	if n != 0 {
		err = errors.New(mgoS.DB.UserIdentityValue + " already registered")
		json.NewEncoder(w).Encode(err)
		return
	}

	old, err := mgoS.DB.ReadId("users", user.ID)

	if err != nil {
		json.NewEncoder(w).Encode(err)
		return
	}

	new := make(map[string]string)

	new[mgoS.DB.UserIdentityValue] = `"` + newValue + `"`

	err = mgoS.DB.UpdateValue("users", old, new)
	if err != nil {
		json.NewEncoder(w).Encode(err)
		return
	}

	json.NewEncoder(w).Encode(mgoS.DB.UserIdentityValue + " changed")
}

//ChangePassword receive a new identity value, checks if exist and changes the old value to the new one
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r)
	newPass := r.PostFormValue("password")
	oldPassword := r.PostFormValue("oldPassword")

	old, err := mgoS.DB.ReadId("users", user.ID)

	if err != nil {
		json.NewEncoder(w).Encode(err)
		return
	}
	salt := strings.Trim(old["password salt"], `"`)

	oldHashCheck := mgoS.GenerateHash(oldPassword, salt)

	if !strings.EqualFold(oldHashCheck, strings.Trim(old["password hash"], `"`)) {
		err = errors.New("incorrect password")
		json.NewEncoder(w).Encode(err)
		return
	}

	new := make(map[string]string)
	new["password hash"] = `"` + mgoS.GenerateHash(newPass, salt) + `"`

	err = mgoS.DB.UpdateValue("users", old, new)
	if err != nil {
		json.NewEncoder(w).Encode(err)
		return
	}

	json.NewEncoder(w).Encode("password changed")
}

//GetUser retorna o ID do usuário a partir do Cookie recebido na reuisição
func GetUser(r *http.Request) (user mgoS.User) {
	if cookie, err := r.Cookie("session"); err == nil {
		if err = Cookie.Decode("session", cookie.Value, &user); err == nil {
			return user
		}
	}
	return user
}
