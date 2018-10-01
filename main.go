package main

import (
	"encoding/json"
	"fmt"

	"github.com/icrowley/fake"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/vk"
	"log"
	"net/http"
	"strconv"
	"strings"
	"io/ioutil"
	"time"
)

const (
	PORT = ":8080"
	APP_ID     = "6707792"
	APP_KEY    = "gQuY2Y2aFVdy9tsIwOAL"
	//APP_SECRET = []byte("fdba0e9ffdba0e9ffdba0e9fc8fddc54cfffdbafdba0e9fa60b49899f33652ed2c03c5f")
	APP_DISPLAY = "page"
	APP_REDIRECT = "http://127.0.0.1" + PORT
)

var (
	APP_SECRET = "fdba0e9ffdba0e9ffdba0e9fc8fddc54cfffdbafdba0e9fa60b49899f33652ed2c03c5f"
	AUTH_URL    = fmt.Sprintf("https://oauth.vk.com/authorize?client_id=%s&display=%s&redirect_uri=%s",APP_ID, APP_DISPLAY,APP_REDIRECT)
	API_URL = ""
)

type MyError struct {
	//Code int `json:"ErrorCode"`
	What string `json:"What"`
}

func generateError(err MyError) []byte {
	result, e := json.Marshal(&err)
	if e != nil {
		log.Fatal("Error while MarshalJson while generating Error")
	}
	return result
}

type User struct {
	Id int `json:"user_id,omitempty"`
	Login string `json:"login"`
	Password string `json:"-"`
	Email    string `json:"email"`
	Score    int  `json:"score"`
}




var users []User = make([]User,0,20)

type BD struct {
	users []User
	lastid int
}

var dataBase BD =BD{make([]User,0,20),0}

func (bd *BD) saveUser(u User) {
	bd.users = append(bd.users, u)
	bd.lastid++
}

func (bd *BD) getUserByEmail(email string) (User,bool){
	for _ , u := range bd.users {
		if u.Email == email {
			return u,true
		}
	}
	return User{},false
}

func (bd *BD) getUserByLogin(login string) (User,bool){
	for _ , u := range bd.users {
		if u.Login == login {
			return u,true
		}
	}
	return User{},false
}

func (bd *BD) getUserByID(id int) (User,bool){
	for _ , u := range bd.users {
		if u.Id == id {
			return u,true
		}
	}
	return User{},false
}

func (db *BD) generateUsers(num int){
	for i:= 0 ; i < num; i++ {
		score,_ := strconv.Atoi(fake.DigitsN(8))

		u := User{db.lastid,fake.FirstName(),fake.SimplePassword(),fake.EmailAddress(),score}
		db.saveUser(u)
	}
	//for _,v := range(users) {
	//	fmt.Println(v)
	//}
}

func init(){
	dataBase.generateUsers(20)
}

func main() {
	//generateUsers(20)

	fmt.Println("AUTH_URL:",AUTH_URL)

	http.HandleFunc("/",func(w http.ResponseWriter, r *http.Request){
		w.Write([]byte("this is backend server API\n"))
		//fmt.Fprintf(w,"<a href=%s>click</a>",AUTH_URL)
		//http.Redirect(w,r,AUTH_URL,http.StatusSeeOther)

	})

	http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request){

		w.Header().Set("content-type", "application/json")
		switch r.Method {
		case http.MethodGet:

			slice := make([]User, 0, 20)
			for _, val := range dataBase.users {
				slice = append(slice, val)
			}
			resp, _ := json.Marshal(&slice)

			w.Write(resp)
		case http.MethodPost:

			body, err := ioutil.ReadAll(r.Body)

			if err != nil {
				log.Println("error while reading body in /user")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var u struct{
				Login string `json:"login"`
				Password string `json:"password"`
				Email    string `json:"email"`
			}
			err = json.Unmarshal(body, &u)
			if _,exist := dataBase.getUserByLogin(u.Login); exist {
				w.Write(generateError(MyError{"User already exist"}))
				return
			}
			var user User = User{Id:dataBase.lastid, Login:u.Login,Email: u.Email, Password: u.Password, Score: 0}
			dataBase.saveUser(user)

			res , err := json.Marshal(&user)
			if err != nil{
				log.Println("error while Marshaling in /user")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(res)
		}
	})

	http.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		switch r.Method{
		case http.MethodGet:
			_, err := r.Cookie("session_id")
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodPost:
			body, err := ioutil.ReadAll(r.Body)

			if err != nil {
				log.Println("error while reading body in /session")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			var u User
			err = json.Unmarshal(body, &u)
			dbUser,exist := dataBase.getUserByLogin(u.Login)
			if !exist {
				w.Write(generateError(MyError{"User does not exist"}))
				return
			}
			if dbUser.Password != u.Password{
				w.Write(generateError(MyError{"wrong password"}))
				return
			}
			cookie := http.Cookie{
				Name: "session_id",
				Value: u.Login+"testCookie"+u.Password,
				Expires:time.Now().Add(30*24*time.Hour),
				HttpOnly: false,
			}
			http.SetCookie(w,&cookie)
			w.WriteHeader(http.StatusOK)
		}

	})

	http.HandleFunc("/user/", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("content-type", "application/json")
		url := r.URL.Path
		url = strings.Trim(url,"/user/")
		id,err := strconv.Atoi(url)
		if err != nil {
			w.Write(generateError(MyError{"Bad URL"}))
			w.WriteHeader(http.StatusBadRequest)
		}
		u,exist := dataBase.getUserByID(id)
		if !exist {
			w.Write(generateError(MyError{"user does not exist"}))
			return
		}
		user,err := json.Marshal(&u)
		if err != nil {
			log.Println("error while Marshaling in /user/")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(user)
	})


	http.HandleFunc("/vkapi", func(w http.ResponseWriter, r *http.Request){
		//http.Redirect(w,r,AUTH_URL,http.StatusSeeOther)

		ctx := r.Context()
		code := r.FormValue("code")
		conf := oauth2.Config{
			ClientID:     APP_ID,
			ClientSecret: APP_KEY,
			RedirectURL:  APP_REDIRECT,
			Endpoint:     vk.Endpoint,
		}

		token, err := conf.Exchange(ctx, code)
		if err != nil {
			log.Println("cannot exchange")
			log.Println(err)
			return
		}

		client := conf.Client(ctx, token)
		resp, err := client.Get(fmt.Sprintf(API_URL, token.AccessToken))
		if err != nil {
			log.Println("cannot request data")
			log.Println(err)
			return
		}
		defer resp.Body.Close()
	})
	fmt.Println("starting server on http://127.0.0.1:8080")



	http.ListenAndServe(":8080", nil)
}


// curl -X POST -d "email=123@mail.ru&passowrd=123" http://127.0.0.1:8080/user
// curl -X POST -d "email=123@mail.ru&passowrd=123" http://127.0.0.1:8080/session
