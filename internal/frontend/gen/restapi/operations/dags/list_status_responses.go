// Code generated by go-swagger; DO NOT EDIT.

package dags

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/dagu-org/dagu/internal/frontend/gen/models"
)

// ListStatusOKCode is the HTTP code returned for type ListStatusOK
const ListStatusOKCode int = 200

/*
ListStatusOK A successful response.

swagger:response listStatusOK
*/
type ListStatusOK struct {

	/*
	  In: Body
	*/
	Payload *models.ListStatusResponse `json:"body,omitempty"`
}

// NewListStatusOK creates ListStatusOK with default headers values
func NewListStatusOK() *ListStatusOK {

	return &ListStatusOK{}
}

// WithPayload adds the payload to the list status o k response
func (o *ListStatusOK) WithPayload(payload *models.ListStatusResponse) *ListStatusOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the list status o k response
func (o *ListStatusOK) SetPayload(payload *models.ListStatusResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ListStatusOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*
ListStatusDefault Generic error response.

swagger:response listStatusDefault
*/
type ListStatusDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.APIError `json:"body,omitempty"`
}

// NewListStatusDefault creates ListStatusDefault with default headers values
func NewListStatusDefault(code int) *ListStatusDefault {
	if code <= 0 {
		code = 500
	}

	return &ListStatusDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the list status default response
func (o *ListStatusDefault) WithStatusCode(code int) *ListStatusDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the list status default response
func (o *ListStatusDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the list status default response
func (o *ListStatusDefault) WithPayload(payload *models.APIError) *ListStatusDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the list status default response
func (o *ListStatusDefault) SetPayload(payload *models.APIError) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ListStatusDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
