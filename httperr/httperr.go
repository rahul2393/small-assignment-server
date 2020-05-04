package httperr

import (
"encoding/json"
"errors"
"fmt"
"net/http"
)

const (
	internalMessage   = "There was a problem with the system. If the problem persists contact the administrator."
	jsonEncodingError = `{"statusCode":500,"message":"There was a problem with the system.  If the problem persists contact the administrator.","error":"json response encoding error"}`
)

// Error is a error wrapper with a status code and user message.  Errors can
// be written out as JSON using the Write method.
type Error struct {
	// StatusCode is the code response code such as 200 for ok or 500 for internal
	// server error
	StatusCode int `json:"statusCode"`
	// Message is a text representation of the error
	Message string `json:"message"`
	// Err is a text representation of the error
	Err string `json:"error"`
}

// New creates an Error with the given code, message, and err.
func New(code int, message string, err error) Error {
	return Error{
		StatusCode: code,
		Message:    message,
		Err:        fmt.Sprintf("%+v", err),
	}
}

// NewInternal creates an Error with a 500 http status and default user message.
func NewInternal(err error) Error {
	if err == nil {
		err = errors.New("")
	}
	return New(http.StatusInternalServerError, err.Error(), err)
}

// NewNotFound creates an Error with a 404 http status and specified user message.
func NewNotFound(err error, whatsNotFound string) Error {
	if err == nil {
		err = errors.New("")
	}
	return New(http.StatusNotFound, fmt.Sprintf("%s not found", whatsNotFound), err)
}

// NewBadRequest creates an Error with a 400 http status and specified user message.
func NewBadRequest(err error, whatTheUserDidWrong string) Error {
	if err == nil {
		err = errors.New("")
	}
	return New(http.StatusBadRequest, whatTheUserDidWrong, err)
}

// Error implements the error interface.
func (e Error) Error() string {
	return e.Err
}

// Write writes out the error to the writer. If the error isn't an httperr.Error
// it creates a 500 error.
func Write(w http.ResponseWriter, err error) {
	e, ok := err.(Error)
	if !ok {
		e = NewInternal(err)
	}
	w.WriteHeader(e.StatusCode)
	if err := json.NewEncoder(w).Encode(e); err != nil {
		http.Error(w, jsonEncodingError, http.StatusInternalServerError)
	}
}
