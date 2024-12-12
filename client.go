package client

import (
	"bytes"
	"errors"
	"fmt"
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

func Do(uid uint64, api string, param any, type_ string) (res []byte, err error) {
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

	request, err := http.NewRequest("POST", u.String(), &buffer)
	if err != nil {
		log.Println(err)
		return
	}
	request.Header.Set("user-id", strconv.FormatUint(uid, 10))
	response, err := client.Do(request)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("status code: %d", http.StatusOK)
		log.Println(err)
		return
	}
	if res, err = io.ReadAll(response.Body); err != nil {
		log.Println(err)
		return
	}

	return
}
