// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/yohamta/dagu/service/frontend/models"
)

// CreateWorkflowOKCode is the HTTP code returned for type CreateWorkflowOK
const CreateWorkflowOKCode int = 200

/*
CreateWorkflowOK A successful response.

swagger:response createWorkflowOK
*/
type CreateWorkflowOK struct {

	/*
	  In: Body
	*/
	Payload *models.CreateWorkflowResponse `json:"body,omitempty"`
}

// NewCreateWorkflowOK creates CreateWorkflowOK with default headers values
func NewCreateWorkflowOK() *CreateWorkflowOK {

	return &CreateWorkflowOK{}
}

// WithPayload adds the payload to the create workflow o k response
func (o *CreateWorkflowOK) WithPayload(payload *models.CreateWorkflowResponse) *CreateWorkflowOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the create workflow o k response
func (o *CreateWorkflowOK) SetPayload(payload *models.CreateWorkflowResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *CreateWorkflowOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

/*
CreateWorkflowDefault Generic error response.

swagger:response createWorkflowDefault
*/
type CreateWorkflowDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.APIError `json:"body,omitempty"`
}

// NewCreateWorkflowDefault creates CreateWorkflowDefault with default headers values
func NewCreateWorkflowDefault(code int) *CreateWorkflowDefault {
	if code <= 0 {
		code = 500
	}

	return &CreateWorkflowDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the create workflow default response
func (o *CreateWorkflowDefault) WithStatusCode(code int) *CreateWorkflowDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the create workflow default response
func (o *CreateWorkflowDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the create workflow default response
func (o *CreateWorkflowDefault) WithPayload(payload *models.APIError) *CreateWorkflowDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the create workflow default response
func (o *CreateWorkflowDefault) SetPayload(payload *models.APIError) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *CreateWorkflowDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
