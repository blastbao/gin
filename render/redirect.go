// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package render

import (
	"fmt"
	"net/http"
)



// HTTP 协议的重定向响应的状态码为 3xx 。
// 浏览器在接收到重定向响应的时候，会采用该响应提供的新的 URL ，并立即进行加载；
// 大多数情况下，除了会有一小部分性能损失之外，重定向操作对于用户来说是不可见的。
// 不同类型的重定向映射可以划分为三个类别：永久重定向、临时重定向和特殊重定向。
//
// reference: 
// 	1. https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Redirections
// 	2. https://colobu.com/2017/04/19/go-http-redirect/

// Redirect contains the http request reference and redirects status code and location.
type Redirect struct {
	Code     int
	Request  *http.Request
	Location string
}

// Render (Redirect) redirects the http request to new location and writes redirect response.
func (r Redirect) Render(w http.ResponseWriter) error {
	// todo(thinkerou): go1.6 not support StatusPermanentRedirect(308)
	// when we upgrade go version we can use http.StatusPermanentRedirect
	if (r.Code < 300 || r.Code > 308) && r.Code != 201 {
		panic(fmt.Sprintf("Cannot redirect with status code %d", r.Code))
	}

	http.Redirect(w, r.Request, r.Location, r.Code)
	return nil
}

// WriteContentType (Redirect) don't write any ContentType.
func (r Redirect) WriteContentType(http.ResponseWriter) {}
