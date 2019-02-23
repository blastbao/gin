// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package binding

import "net/http"

const defaultMemory = 32 * 1024 * 1024

type formBinding struct{}
type formPostBinding struct{}
type formMultipartBinding struct{}


func (formBinding) Name() string {
	return "form"
}

func (formBinding) Bind(req *http.Request, obj interface{}) error {

	//parse url and "application/x-www-form-urlencoded" body params, saved both in req.PostForm and req.Form .
	if err := req.ParseForm(); err != nil {
		return err
	}

	//parse "multipart/form-data" body params, saved both in req.MultipartForm and req.Form .
	if err := req.ParseMultipartForm(defaultMemory); err != nil {
		if err != http.ErrNotMultipart {
			return err
		}
	}

	// parse req.Form and convert it then save in obj
	if err := mapForm(obj, req.Form); err != nil {
		return err
	}
	return validate(obj)
}

func (formPostBinding) Name() string {
	return "form-urlencoded"
}

func (formPostBinding) Bind(req *http.Request, obj interface{}) error {
	//parse url and "application/x-www-form-urlencoded" body params, saved in r.PostForm.
	if err := req.ParseForm(); err != nil {
		return err
	}
	if err := mapForm(obj, req.PostForm); err != nil {
		return err
	}
	return validate(obj)
}

func (formMultipartBinding) Name() string {
	return "multipart/form-data"
}

func (formMultipartBinding) Bind(req *http.Request, obj interface{}) error {
	//parse "multipart/form-data" body params, saved in r.MultipartForm.
	if err := req.ParseMultipartForm(defaultMemory); err != nil {
		return err
	}
	if err := mapForm(obj, req.MultipartForm.Value); err != nil {
		return err
	}
	return validate(obj)
}
