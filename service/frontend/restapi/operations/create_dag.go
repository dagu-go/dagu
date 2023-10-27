// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"context"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// CreateDagHandlerFunc turns a function with the right signature into a create dag handler
type CreateDagHandlerFunc func(CreateDagParams) middleware.Responder

// Handle executing the request and returning a response
func (fn CreateDagHandlerFunc) Handle(params CreateDagParams) middleware.Responder {
	return fn(params)
}

// CreateDagHandler interface for that can handle valid create dag params
type CreateDagHandler interface {
	Handle(CreateDagParams) middleware.Responder
}

// NewCreateDag creates a new http.Handler for the create dag operation
func NewCreateDag(ctx *middleware.Context, handler CreateDagHandler) *CreateDag {
	return &CreateDag{Context: ctx, Handler: handler}
}

/*
	CreateDag swagger:route POST /dags createDag

Creates a new DAG.
*/
type CreateDag struct {
	Context *middleware.Context
	Handler CreateDagHandler
}

func (o *CreateDag) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewCreateDagParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}

// CreateDagBody create dag body
//
// swagger:model CreateDagBody
type CreateDagBody struct {

	// action
	// Required: true
	Action *string `json:"action"`

	// value
	// Required: true
	Value *string `json:"value"`
}

// Validate validates this create dag body
func (o *CreateDagBody) Validate(formats strfmt.Registry) error {
	var res []error

	if err := o.validateAction(formats); err != nil {
		res = append(res, err)
	}

	if err := o.validateValue(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (o *CreateDagBody) validateAction(formats strfmt.Registry) error {

	if err := validate.Required("body"+"."+"action", "body", o.Action); err != nil {
		return err
	}

	return nil
}

func (o *CreateDagBody) validateValue(formats strfmt.Registry) error {

	if err := validate.Required("body"+"."+"value", "body", o.Value); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this create dag body based on context it is used
func (o *CreateDagBody) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (o *CreateDagBody) MarshalBinary() ([]byte, error) {
	if o == nil {
		return nil, nil
	}
	return swag.WriteJSON(o)
}

// UnmarshalBinary interface implementation
func (o *CreateDagBody) UnmarshalBinary(b []byte) error {
	var res CreateDagBody
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*o = res
	return nil
}
