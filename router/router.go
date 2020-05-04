package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/urfave/negroni"
	"github.com/rahul2393/small-assignment-server/handler"
	"github.com/rahul2393/small-assignment-server/models/acct"
	"github.com/rahul2393/small-assignment-server/models/model"
	"github.com/rahul2393/small-assignment-server/mware"
)

const (
	// GET represents the HTTP GET Method
	GET = "GET"
	// POST represents the HTTP POST Method
	POST = "POST"
	// PUT represents the HTTP PUT Method
	PUT = "PUT"
	// DELETE represents the HTTP DELETE Method
	DELETE = "DELETE"
)

func SetupRouters() http.Handler {
	// create base router
	r := mux.NewRouter().StrictSlash(true)

	// health check handler
	r.HandleFunc("/", serverUp).Methods("GET")

	post(r, "/signup", handler.SignUp())
	post(r, "/login", handler.Login())
	subRouter := createSubRouter(r, "/api", mware.UserAuth())

	get(subRouter, "/signout", handler.SignOut())
	post(subRouter, "/user/{id}/resetPassword", handler.ResetPassword())
	post(subRouter, "/createUser", handler.CreateUser())
	get(subRouter, "/user/{id}/updateGroup/{groupId}", handler.UpdateUserGroup())

	addRUD(subRouter, "/users", &acct.User{})
	addCRUD(subRouter, "/meals", &model.Meal{})

	// add middleware common to all handlers
	n := negroni.New(
		negroni.NewRecovery(),
		mware.JSONContentType{},
		cors.New(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		}),
	)
	n.UseHandler(r)
	return n
}

func createSubRouter(mainRouter *mux.Router, prefix string, middleware ...negroni.Handler) *mux.Router {
	newRouter := mux.NewRouter()
	middleware = append(middleware, negroni.Wrap(newRouter))
	n := negroni.New(middleware...)
	mainRouter.PathPrefix(prefix).Handler(n)
	return newRouter.PathPrefix(prefix).Subrouter()
}

var serverUp http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"up":true}`)
}

func get(r *mux.Router, path string, h http.Handler) {
	r.Handle(path, h).Methods(GET)
}

func post(r *mux.Router, path string, h http.Handler) {
	r.Handle(path, h).Methods(POST)
}

func put(r *mux.Router, path string, h http.Handler) {
	r.Handle(path, h).Methods(PUT)
}

func delete(r *mux.Router, path string, h http.Handler) {
	r.Handle(path, h).Methods(DELETE)
}

func addRUD(r *mux.Router, path string, m mware.MergeModel) {
	r.Handle(path, mware.GetAll(m)).Methods(GET)
	r.Handle(path+"/{id}", mware.GetByID(m)).Methods(GET)
	r.Handle(path+"/{id}", mware.UpdateByID(m)).Methods(PUT)
	r.Handle(path+"/{id}", mware.DeleteByID(m)).Methods(DELETE)
}

func addCRUD(r *mux.Router, path string, m mware.MergeModel) {
	r.Handle(path, mware.GetAll(m)).Methods(GET)
	r.Handle(path+"/{id}", mware.GetByID(m)).Methods(GET)
	r.Handle(path, mware.Create(m)).Methods(POST)
	r.Handle(path+"/{id}", mware.UpdateByID(m)).Methods(PUT)
	r.Handle(path+"/{id}", mware.DeleteByID(m)).Methods(DELETE)
}
