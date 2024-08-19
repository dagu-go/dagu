// Code generated by go-swagger; DO NOT EDIT.

package dags

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
)

// NewListStatusesParams creates a new ListStatusesParams object
//
// There are no default values defined in the spec.
func NewListStatusesParams() ListStatusesParams {

	return ListStatusesParams{}
}

// ListStatusesParams contains all the bound params for the list statuses operation
// typically these are obtained from a http.Request
//
// swagger:parameters listStatuses
type ListStatusesParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*
	  Required: true
	  In: query
	*/
	Date string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewListStatusesParams() beforehand.
func (o *ListStatusesParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	qs := runtime.Values(r.URL.Query())

	qDate, qhkDate, _ := qs.GetOK("date")
	if err := o.bindDate(qDate, qhkDate, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindDate binds and validates parameter Date from query.
func (o *ListStatusesParams) bindDate(rawData []string, hasKey bool, formats strfmt.Registry) error {
	if !hasKey {
		return errors.Required("date", "query", rawData)
	}
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// AllowEmptyValue: false

	if err := validate.RequiredString("date", "query", raw); err != nil {
		return err
	}
	o.Date = raw

	return nil
}