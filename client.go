package client

import (
	"bytes"
	"errors"
	"github.com/clong1995/go-encipher/gob"
	"github.com/clong1995/go-encipher/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var client *http.Client

func init() {
	client = &http.Client{
		Timeout: 10 * time.Second,
	}
}

const (
	JSON = iota
	GOB
	BYTES
)

// Do 发起请求
// type_: 对方接口接收数据的类型JSON,GOB,BYTES
// res T: 按照约定，对方返回的要和请求的类型相同,T范型会自序列化为对应的类型,当type_为BYTES,范型必须为[]byte
func Do[T any](uid int64, api, method string, param any, type_ int, header ...map[string]string) (res T, err error) {
	u, err := url.Parse(api)
	if err != nil {
		log.Println(err)
		return
	}

	var buffer *bytes.Buffer

	if param != nil {
		if method == http.MethodGet {
			options := param.(map[string]string)
			q := u.Query()
			for k, v := range options {
				q.Set(k, v)
			}
			u.RawQuery = q.Encode()
		} else {
			if type_ == JSON {
				if err = json.Encode(param, buffer); err != nil {
					log.Println(err)
					return
				}
			} else if type_ == GOB {
				if err = gob.Encode(param, buffer); err != nil {
					log.Println(err)
					return
				}
			} else if type_ == BYTES {
				buffer = bytes.NewBuffer(param.([]byte))
			} else {
				err = errors.New("type is required")
				log.Println(err)
				return
			}
		}
	}

	request, err := http.NewRequest(method, u.String(), buffer)
	if err != nil {
		log.Println(err)
		return
	}

	if uid != 0 {
		request.Header.Set("user-id", strconv.FormatInt(uid, 10))
	}

	if len(header) > 0 {
		for k, v := range header[0] {
			request.Header.Set(k, v)
		}
	}

	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		var body []byte
		if body, err = io.ReadAll(response.Body); err != nil {
			log.Println(err)
			return
		}
		err = errors.New(string(body))
		log.Println(err)
		return
	}

	if type_ == JSON {
		if err = json.Decode(response.Body, &res); err != nil {
			log.Println(err)
			return
		}
	} else if type_ == GOB { // gob可以返回任何类型的结果，T 是啥就是啥
		if err = gob.Decode(response.Body, &res); err != nil {
			log.Println(err)
			return
		}
	} else if type_ == BYTES {
		var body any
		if body, err = io.ReadAll(response.Body); err != nil {
			log.Println(err)
			return
		}
		res = body.(T)
	}
	return
}
