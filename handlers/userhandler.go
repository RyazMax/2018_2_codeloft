package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-park-mail-ru/2018_2_codeloft/models"
	"github.com/go-park-mail-ru/2018_2_codeloft/services"
	"github.com/go-park-mail-ru/2018_2_codeloft/validator"
	"github.com/mailru/easyjson"

	"github.com/go-park-mail-ru/2018_2_codeloft/auth"
	"go.uber.org/zap"
)

func generateError(err models.MyError) []byte {
	result, e := json.Marshal(&err)
	if e != nil {
		log.Fatal("Error while MarshalJson while generating Error")
		zap.L().Fatal("Erro while MarshalJson while generating Error",
			zap.Error(e))
	}
	return result
}

func leaders(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Add("XXX_VAASYA", "TAX")
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		myError := models.MyError{URL: r.URL.Path, What: "error while parsing form", Err: err}
		w.Write(generateError(myError))
		zap.L().Info("Parsing error in leaders",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(&myError),
		)

		//w.Write(generateError(models.MyError{r.URL.Path, "error while parsing form", err}))

		return
	}
	var page int
	var pageSize int

	checkParam := func(w http.ResponseWriter, r *http.Request, param string) (int, error) {
		param_str := r.FormValue(param)
		var paramReturn int
		if param_str == "" {
			paramReturn = 0
		} else {
			paramReturn, err = strconv.Atoi(r.FormValue(param))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write(generateError(models.MyError{URL: r.URL.Path, What: "Bad params", Err: err}))

				zap.L().Info("Bad params",
					zap.String("URL", r.URL.Path),
					zap.String("Method", r.Method),
					zap.String("Origin", r.Header.Get("Origin")),
					zap.String("Remote addres", r.RemoteAddr),
					zap.Error(err),
				)

				return -1, err
			}
		}
		return paramReturn, nil
	}

	if page, err = checkParam(w, r, "page"); err != nil {
		return
	}
	if pageSize, err = checkParam(w, r, "page_size"); err != nil {
		return
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 5
	}
	lead := models.GetLeaders(db, page, pageSize)
	resp, _ := json.Marshal(&lead)
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
	return
}

func signUp(w http.ResponseWriter, r *http.Request, db *sql.DB, sm auth.AuthCheckerClient) {
	//var s *models.Session
	//if s = services.GetCookie(r, db); s != nil {
	//	w.WriteHeader(http.StatusConflict)
	//	return
	//}
	//cooka, err := r.Cookie("session_id")
	//if err == nil {
	//	w.WriteHeader(http.StatusConflict)
	//	log.Println("[ERROR] signUp Cookie exist.AlreadyAuth:", cooka.Value)
	//	return
	//}
	cooka, err := r.Cookie("session_id")
	if cooka != nil {
		userid, err := sm.Check(context.Background(), &auth.SessionID{ID: cooka.Value})
		if err == nil {
			fmt.Println("[ERROR] signUp: Already auth.Need logout UserID:", userid.UserID)
			w.WriteHeader(http.StatusConflict)
			return
		}
	}
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		zap.L().Info("error while reading body /user",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	u := models.HelpUser{}
	err = easyjson.Unmarshal(body, &u)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "wrong request format", Err: err}))

		zap.L().Info("wrong request format",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)

		return
	}
	var user models.User
	if exist := user.GetUserByLogin(db, u.Login); exist {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "User already exist", Err: fmt.Errorf("")}))
		return
	}
	err = validator.ValidateEmail(u.Email)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "bad email", Err: err}))

		zap.L().Info("bad email",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)

		return
	}
	err = validator.ValidateLogin(u.Login)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "bad login", Err: err}))

		zap.L().Info("bad login",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)

		return
	}
	err = validator.ValidatePassword(u.Password)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "bad password", Err: err}))

		zap.L().Info("bad password",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)

		return
	}
	user = models.User{Login: u.Login, Email: u.Email, Password: u.Password}
	err = user.AddUser(db)
	if err != nil {
		zap.L().Warn("Can not add user",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)
	}
	res, err := easyjson.Marshal(&user)
	if err != nil {
		zap.L().Info("error while Marshaling in /user",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// SET COOKIE
	cookie := services.GenerateCookie(user.Login)
	if os.Getenv("ENV") == "production" {
		cookie.Secure = true
	}
	session := models.Session{Value: cookie.Value, User_id: user.Id}
	session.AddCookie(db)
	http.SetCookie(w, cookie)

	w.Write(res)
}

func updateUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	_, err := r.Cookie("session_id")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "authorization required", Err: err}))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {

		zap.L().Info("error while reading body in /user",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	u := models.HelpUser{}
	err = easyjson.Unmarshal(body, &u)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "wrong request format", Err: err}))
		return
	}
	err = validator.ValidateLogin(u.Login)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "bad login", Err: err}))
		return
	}
	var user models.User
	if !user.GetUserByLogin(db, u.Login) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "User does not exist", Err: err}))
		return
	}

	if user.Password != u.Password {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "wrong password", Err: fmt.Errorf("")}))
		return
	}
	var newPassword string = user.Password
	var newEmail string = user.Email
	var newScore int64
	if u.NewPassword != "" {
		newPassword = u.NewPassword
	}
	err = validator.ValidatePassword(newPassword)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "bad New password", Err: err}))
		return
	}
	if u.Email != "" {
		newEmail = u.Email
	}
	err = validator.ValidateEmail(newEmail)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "bad New email", Err: err}))
		return
	}
	if u.Score != 0 {
		newScore = u.Score
	}

	newUser := models.User{Id: user.Id, Login: u.Login, Password: newPassword, Email: newEmail, Score: newScore, Lang: user.Lang, Avatar: ""}
	err = newUser.UpdateUser(db)
	zap.L().Info("Can not update user",
		zap.String("URL", r.URL.Path),
		zap.String("Method", r.Method),
		zap.String("Origin", r.Header.Get("Origin")),
		zap.String("Remote addres", r.RemoteAddr),
		zap.String("User", newUser.Login),
		zap.Error(err),
	)
	newUser.UpdateScore(db)

	var result struct {
		Id    int64  `json:"user_id"`
		Login string `json:"login"`
		Email string `json:"email"`
		Score int64  `json:"score"`
	}
	result.Id = newUser.Id
	result.Login = newUser.Login
	result.Email = newUser.Email
	result.Score = newUser.Score
	w.WriteHeader(http.StatusOK)
	res, err := json.Marshal(&result)
	if err != nil {
		zap.L().Info("error while Marshaling in /user",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(res)
}

// UserHandler handler for api/user
type UserHandler struct {
	Db *sql.DB
	Sm auth.AuthCheckerClient
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	switch r.Method {

	case http.MethodGet:
		leaders(w, r, h.Db)
	case http.MethodPost:
		signUp(w, r, h.Db, h.Sm)
	case http.MethodPut:
		updateUser(w, r, h.Db)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

//var UserById = func(w http.ResponseWriter, r *http.Request) {
//	w.Header().Set("content-type", "application/json")
//	url := r.URL.Path
//	url = strings.Trim(url, "/user/")
//	id, err := strconv.Atoi(url)
//	if err != nil {
//		w.WriteHeader(http.StatusBadRequest)
//		w.Write(generateError(MyError{"Bad URL"}))
//		return
//	}
//	u, exist := dataBase.GetUserByID(id)
//	if !exist {
//		w.WriteHeader(http.StatusBadRequest)
//		w.Write(generateError(MyError{"user does not exist"}))
//		return
//	}
//	user, err := json.Marshal(&u)
//	if err != nil {
//		log.Println("error while Marshaling in /user/")
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//	w.Write(user)
//}

// UserById handler for api requests with user id
type UserById struct {
	Db *sql.DB
	Sm auth.AuthCheckerClient
}

func userGet(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	url := r.URL.Path
	url = strings.Trim(url, "/user/")
	id, err := strconv.ParseInt(url, 10, 64)
	if err != nil {
		zap.L().Info("Parsing error",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "Bad URL", Err: err}))
		return
	}
	var u models.User
	if !u.GetUserByID(db, id) {
		w.WriteHeader(http.StatusBadRequest)
		//w.Write(generateError(models.MyError{r.URL.Path,"user does not exist",models.UserDoesNotExist(u.Login)}))
		return
	}
	user, err := json.Marshal(&u)
	if err != nil {
		zap.L().Info("error while Marshaling in /user",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(user)
}

func userDelete(w http.ResponseWriter, r *http.Request, db *sql.DB, sm auth.AuthCheckerClient) {
	url := r.URL.Path
	url = strings.Trim(url, "/user/")
	id, err := strconv.ParseInt(url, 10, 64)
	if err != nil {
		zap.L().Info("Parsing error",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "Bad URL", Err: err}))
		return
	}
	user := &models.User{}
	if !user.GetUserByID(db, id) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//var s *models.Session
	//
	//if s = services.GetCookie(r, db); s == nil {
	//	w.WriteHeader(http.StatusUnauthorized)
	//	return
	//}
	cookie, err := r.Cookie("session_id")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.Println("No cookie header with session_id name", err)
		return
	}
	userid, err := sm.Check(context.Background(), &auth.SessionID{ID: cookie.Value})
	if err != nil {
		fmt.Println("[ERROR] checkAuth:", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if id != userid.UserID {
		w.WriteHeader(http.StatusConflict)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "user id != url id", Err: fmt.Errorf("user_id = %d. url ud = %d", userid.UserID, id)}))
		return
	}

	err = user.DeleteUser(db)
	if err != nil {
		zap.L().Warn("Can not delete user",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.String("User", user.Login),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusNotFound)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "Cant del user", Err: err}))
	}
	w.WriteHeader(http.StatusOK)
}

func (h *UserById) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	switch r.Method {

	case http.MethodGet:
		userGet(w, r, h.Db)
	case http.MethodDelete:
		userDelete(w, r, h.Db, h.Sm)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// UserLang is handler for changing language of user
type UserLang struct {
	Db *sql.DB
	Sm auth.AuthCheckerClient
}

func userUpdateLang(w http.ResponseWriter, r *http.Request, db *sql.DB, sm auth.AuthCheckerClient) {
	//var session *models.Session
	//if session = services.GetCookie(r, db); session == nil {
	//	w.WriteHeader(http.StatusUnauthorized)
	//	return
	//}
	cookie, err := r.Cookie("session_id")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.Println("No cookie header with session_id name", err)
		return
	}
	userid, err := sm.Check(context.Background(), &auth.SessionID{ID: cookie.Value})
	if err != nil {
		fmt.Println("[ERROR] checkAuth in userUpdateLang:", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var user models.User
	if !user.GetUserByID(db, userid.UserID) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "User Does Not Exist in Users table, but exist in session", Err: fmt.Errorf("")}))
		zap.L().Info("User Does Not Exist in Users table, but exist in session",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.String("Session value", cookie.Value),
			zap.Int64("User id", userid.UserID),
		)
		return
	}

	var lang struct {
		Lang string `json:"lang"`
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		zap.S().Infow("Error in lang update", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &lang)
	if err != nil {
		zap.S().Infow("Error in lang update", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//user := models.User{}
	//if !user.GetUserByID(db, session.User_id) {
	//	w.WriteHeader(http.StatusBadRequest)
	//	return
	//}

	err = validator.ValidateLang(lang.Lang)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		zap.S().Infow("Incorrect language", "lang", lang.Lang)
		return
	}
	user.Lang = lang.Lang

	err = user.UpdateLang(db)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		zap.S().Infow("Error in lang update", "err", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	body, err = json.Marshal(user)
	w.Write(body)
}

func (h *UserLang) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	switch r.Method {
	case http.MethodPost:
		userUpdateLang(w, r, h.Db, h.Sm)
	}
}

// UserScore is handler for changing score
type UserScore struct {
	Db *sql.DB
	Sm auth.AuthCheckerClient
}

func userUpdateScore(w http.ResponseWriter, r *http.Request, db *sql.DB, sm auth.AuthCheckerClient) {
	//var session *models.Session
	//if session = services.GetCookie(r, db); session == nil {
	//	w.WriteHeader(http.StatusUnauthorized)
	//	return
	//}
	cookie, err := r.Cookie("session_id")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.Println("No cookie header with session_id name", err)
		return
	}
	userid, err := sm.Check(context.Background(), &auth.SessionID{ID: cookie.Value})
	if err != nil {
		fmt.Println("[ERROR] checkAuth in userUpdateLang:", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var user models.User
	if !user.GetUserByID(db, userid.UserID) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "User Does Not Exist in Users table, but exist in session", Err: fmt.Errorf("")}))
		zap.L().Info("User Does Not Exist in Users table, but exist in session",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.String("Session value", cookie.Value),
			zap.Int64("User id", userid.UserID),
		)
		return
	}

	var score struct {
		Score string `json:"score"`
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		zap.S().Infow("Error in score update", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &score)
	if err != nil {
		zap.S().Infow("Error in score update", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//user := models.User{}
	//if !user.GetUserByID(db, session.User_id) {
	//	w.WriteHeader(http.StatusBadRequest)
	//	return
	//}

	sc, err := strconv.ParseInt(score.Score, 10, 64)
	if err != nil {
		zap.L().Info("Convert Error. Bad Score",
			zap.String("URL", r.URL.Path),
			zap.String("Method", r.Method),
			zap.String("Origin", r.Header.Get("Origin")),
			zap.String("Remote addres", r.RemoteAddr),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(generateError(models.MyError{URL: r.URL.Path, What: "Bad URL", Err: err}))
		return
	}
	user.Score = sc

	err = user.UpdateScore(db)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		zap.S().Infow("Error in lang update", "err", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	body, err = json.Marshal(user)
	w.Write(body)
}

func (h *UserScore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	switch r.Method {
	case http.MethodPost:
		userUpdateScore(w, r, h.Db, h.Sm)
	}
}
