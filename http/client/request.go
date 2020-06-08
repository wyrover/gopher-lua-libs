package http

import (
	"bytes"
	lua "github.com/yuin/gopher-lua"
	"io/ioutil"
	"net/http"
	"net/url"
)

type luaRequest struct {
	*http.Request
}

func checkRequest(L *lua.LState, n int) *luaRequest {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*luaRequest); ok {
		return v
	}
	L.ArgError(1, "http request excepted")
	return nil
}

// http.request(verb, url, body) returns user-data, error
func NewRequest(L *lua.LState) int {
	verb := L.CheckString(1)
	url := L.CheckString(2)
	buffer := &bytes.Buffer{}
	if L.GetTop() > 2 {
		buffer.WriteString(L.CheckString(3))
	}
	httpReq, err := http.NewRequest(verb, url, buffer)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	req := &luaRequest{Request: httpReq}
	req.Request.Header.Set(`User-Agent`, DefaultUserAgent)
	ud := L.NewUserData()
	ud.Value = req
	L.SetMetatable(ud, L.GetTypeMetatable("http_request_ud"))
	L.Push(ud)
	return 1
}

// request:set_basic_auth(username, password)
func SetBasicAuth(L *lua.LState) int {
	req := checkRequest(L, 1)
	user, passwd := L.CheckAny(2).String(), L.CheckAny(3).String()
	req.SetBasicAuth(user, passwd)
	return 0
}

// request:header_set(key, value)
func HeaderSet(L *lua.LState) int {
	req := checkRequest(L, 1)
	key, value := L.CheckAny(2).String(), L.CheckAny(3).String()
	req.Header.Set(key, value)
	return 0
}

// DoRequest lua http_client_ud:do_request()
// http_client_ud:do_request(http_request_ud) returns (response, error)
//    response: {
//      code = http_code (200, 201, ..., 500, ...),
//      body = string
//      headers = table
//    }
func DoRequest(L *lua.LState) int {
	client := checkClient(L)
	req := checkRequest(L, 2)

	response, err := client.DoRequest(req.Request)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer response.Body.Close()
	headers := L.NewTable()

	for k, v := range response.Header {
		if len(v) > 0 {
			headers.RawSetString(k, lua.LString(v[0]))
		}
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	result := L.NewTable()
	L.SetField(result, `code`, lua.LNumber(response.StatusCode))
	L.SetField(result, `body`, lua.LString(string(data)))
	L.SetField(result, `headers`, headers)
	L.Push(result)
	return 1
}

func GetCookie(L *lua.LState) int {
	client := checkClient(L)
	req := L.CheckString(2)

	url,_ := url.Parse(req)
	cookies := client.Client.Jar.Cookies(url)

	result := L.NewTable()

	if cookies == nil {
		L.Push(lua.LNil)
		return 1
	}

	for _,v := range(cookies){
		L.SetField(result, v.Name, lua.LString(v.Value))
	}

	L.Push(result)
	return 1
}
