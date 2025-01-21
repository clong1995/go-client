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

func Do[T any](uid int64, api, method string, param any, type_ string, header ...map[string]string) (res T, err error) {
	u, err := url.Parse(api)
	if err != nil {
		log.Println(err)
		return
	}

	var buffer bytes.Buffer

	if param != nil {
		if type_ == "JSON" {
			if err = json.Encode(param, &buffer); err != nil {
				log.Println(err)
				return
			}
		} else if type_ == "GOB" {
			if err = gob.Encode(param, &buffer); err != nil {
				log.Println(err)
				return
			}
		} else {
			err = errors.New("type is required")
			log.Println(err)
			return
		}
	}

	request, err := http.NewRequest(method, u.String(), &buffer)
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

	if type_ == "JSON" {
		if err = json.Decode(response.Body, &res); err != nil {
			log.Println(err)
			return
		}
	} else if type_ == "GOB" {
		if err = gob.Decode(response.Body, &res); err != nil {
			log.Println(err)
			return
		}
	}
	return
}
