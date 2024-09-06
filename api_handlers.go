// generated with codegen
package main

import "errors"
import "fmt"
import "net/http"
import "net/url"
import "slices"
import "strconv"
import "strings"

func (p *ProfileParams) fillAndValidate(v url.Values) (err error) {
	 var param string

	param = v.Get("login")

	if param == "" {
		return fmt.Errorf("login must me not empty")
	}
	p.Login = param
	return
}
func (p *CreateParams) fillAndValidate(v url.Values) (err error) {
	 var param string

	param = v.Get("login")

	if param == "" {
		return fmt.Errorf("login must me not empty")
	}
	p.Login = param

	if len(p.Login) < 10 {
		return fmt.Errorf("login len must be >= 10")
	}

	param = v.Get("full_name")
	p.Name = param

	param = v.Get("status")

	if param == "" {
		p.Status = "user"
	} else {
		p.Status = param
	}

	enums := strings.Split("user|moderator|admin", "|")
	if !slices.Contains(enums, p.Status) {
		return fmt.Errorf("status must be one of [%s]", strings.Join(enums, ", "))
	}

	param = v.Get("age")
	intParam, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("age must be int")
	}
	p.Age = intParam

	if p.Age < 0 {
		return fmt.Errorf("age must be >= 0")
	}

	if p.Age > 128 {
		return fmt.Errorf("age must be <= 128")
	}
	return
}
func (srv *MyApi) wrapperProfile(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()
	var params ProfileParams
	err := params.fillAndValidate(r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeResponse(w, err.Error(), nil)
		return
	}

	res, err := srv.Profile(r.Context(), params)
	if err != nil {
		var apiError ApiError
		if errors.As(err, &apiError) {
			w.WriteHeader(apiError.HTTPStatus)
			writeResponse(w, apiError.Error(), nil)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			writeResponse(w, err.Error(), nil)
		}
		return
	}

	writeResponse(w, "", res)
}
func (srv *MyApi) wrapperCreate(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotAcceptable)
		writeResponse(w, "bad method", nil)
		return
	}

	if r.Header.Get("X-Auth") != "100500" {
		w.WriteHeader(http.StatusForbidden)
		writeResponse(w, "unauthorized", nil)
		return
	}

	r.ParseForm()
	var params CreateParams
	err := params.fillAndValidate(r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeResponse(w, err.Error(), nil)
		return
	}

	res, err := srv.Create(r.Context(), params)
	if err != nil {
		var apiError ApiError
		if errors.As(err, &apiError) {
			w.WriteHeader(apiError.HTTPStatus)
			writeResponse(w, apiError.Error(), nil)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			writeResponse(w, err.Error(), nil)
		}
		return
	}

	writeResponse(w, "", res)
}
func (p *OtherCreateParams) fillAndValidate(v url.Values) (err error) {
	 var param string

	param = v.Get("username")

	if param == "" {
		return fmt.Errorf("username must me not empty")
	}
	p.Username = param

	if len(p.Username) < 3 {
		return fmt.Errorf("username len must be >= 3")
	}

	param = v.Get("account_name")
	p.Name = param

	param = v.Get("class")

	if param == "" {
		p.Class = "warrior"
	} else {
		p.Class = param
	}

	enums := strings.Split("warrior|sorcerer|rouge", "|")
	if !slices.Contains(enums, p.Class) {
		return fmt.Errorf("class must be one of [%s]", strings.Join(enums, ", "))
	}

	param = v.Get("level")
	intParam, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("level must be int")
	}
	p.Level = intParam

	if p.Level < 1 {
		return fmt.Errorf("level must be >= 1")
	}

	if p.Level > 50 {
		return fmt.Errorf("level must be <= 50")
	}
	return
}
func (srv *OtherApi) wrapperCreate(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotAcceptable)
		writeResponse(w, "bad method", nil)
		return
	}

	if r.Header.Get("X-Auth") != "100500" {
		w.WriteHeader(http.StatusForbidden)
		writeResponse(w, "unauthorized", nil)
		return
	}

	r.ParseForm()
	var params OtherCreateParams
	err := params.fillAndValidate(r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeResponse(w, err.Error(), nil)
		return
	}

	res, err := srv.Create(r.Context(), params)
	if err != nil {
		var apiError ApiError
		if errors.As(err, &apiError) {
			w.WriteHeader(apiError.HTTPStatus)
			writeResponse(w, apiError.Error(), nil)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			writeResponse(w, err.Error(), nil)
		}
		return
	}

	writeResponse(w, "", res)
}
func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		h.wrapperProfile(w, r)
	case "/user/create":
		h.wrapperCreate(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		writeResponse(w, "unknown method", nil)
	}
}
func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		h.wrapperCreate(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		writeResponse(w, "unknown method", nil)
	}
}
