package client

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/clong1995/go-encipher/gob"
	"github.com/clong1995/go-encipher/json"
)

// client 是一个可复用的HTTP客户端，具有10秒的超时设置。
var client = &http.Client{
	Timeout: 10 * time.Second,
}

// 定义支持的内容类型常量
const (
	NIL      = iota
	JSON     // JSON 格式
	GOB      // GOB 格式
	BYTES    // 原始字节流
	FORMDATA // form-data
)

// Do 发起一个HTTP请求。
func Do[T any](uid int64, api, method string, param any, reqContentType, respContentType int, header ...map[string]any) (T, error) {
	var res T // 初始化响应结果变量

	// 1. 解析API URL
	u, err := url.Parse(api)
	if err != nil {
		return res, errors.WithStack(err)
	}

	body := &bytes.Buffer{} // 请求体
	var writer *multipart.Writer

	// 2. 处理请求参数
	if param != nil {
		if method == http.MethodGet {
			// 对于GET请求，将参数编码到URL查询字符串中
			q := u.Query()
			switch p := param.(type) {
			case map[string]any:
				for k, v := range p {
					q.Set(k, fmt.Sprintf("%v", v))
				}
			case map[string]string:
				for k, v := range p {
					q.Set(k, v)
				}
			default:
				return res, errors.New("for GET requests, param must be map[string]any or map[string]string")
			}
			u.RawQuery = q.Encode()
		} else {
			// 对于非GET请求（如POST, PUT等），将参数编码到请求体中
			//buf := new(bytes.Buffer)
			switch reqContentType {
			case JSON:
				buf := new(bytes.Buffer)
				// 使用JSON编码
				if err = json.Encode(param, buf); err != nil {
					return res, errors.WithStack(err)
				}
				body = buf
			case GOB:
				buf := new(bytes.Buffer)
				// 使用GOB编码
				if err = gob.Encode(param, buf); err != nil {
					return res, errors.WithStack(err)
				}
				body = buf
			case BYTES:
				if b, ok := param.([]byte); ok {
					body = bytes.NewBuffer(b)
				} else {
					return res, errors.New("for BYTES content type, param must be []byte")
				}
			case FORMDATA:
				if r, ok := param.(map[string]string); ok {
					writer = multipart.NewWriter(body)
					defer func() {
						_ = writer.Close()
					}()
					for key, value := range r {
						if err = writer.WriteField(key, value); err != nil {
							return res, errors.WithStack(err)
						}
					}
					// 关闭 writer，此时会写入最终的 boundary 结束符
					if err = writer.Close(); err != nil {
						return res, errors.WithStack(err)
					}
				} else {
					return res, errors.New("for FORMDATA content type, param must be map[string]string")
				}
			case NIL:
				// 什么都不做，body 保持为 nil
			default:
				return res, errors.New("unsupported content type")
			}
		}
	}

	// 3. 创建HTTP请求
	request, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return res, errors.WithStack(err)
	}

	// 4. 设置请求头
	// 根据内容类型设置Content-Type
	if body != nil {
		switch reqContentType {
		case JSON:
			request.Header.Set("Content-Type", "application/json")
		case GOB, BYTES:
			request.Header.Set("Content-Type", "application/octet-stream")
		case FORMDATA:
			if writer != nil {
				request.Header.Set("Content-Type", writer.FormDataContentType())
			}
		case NIL:
		}
	}

	// 根据期望的响应内容类型设置 Accept 头
	switch respContentType {
	case JSON:
		request.Header.Set("Accept", "application/json")
	case GOB, BYTES:
		request.Header.Set("Accept", "application/octet-stream")
	case NIL:
	default:
	}

	// 如果提供了用户ID，则设置user-id请求头
	if uid != 0 {
		request.Header.Set("user-id", strconv.FormatInt(uid, 10))
	}

	// 设置自定义的额外请求头
	if len(header) > 0 && header[0] != nil {
		for k, v := range header[0] {
			request.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	// 5. 发送HTTP请求
	response, err := client.Do(request)
	if err != nil {
		return res, errors.WithStack(err)
	}

	// 确保响应体在函数结束时关闭，并排空以便复用连接
	defer func() {
		_, _ = io.Copy(io.Discard, response.Body)
		response.Body.Close()
	}()

	// 6. 检查HTTP响应状态码
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return res, errors.WithStack(readErr)
		}
		return res, errors.Errorf("http request failed with status %d: \nbody:%s", response.StatusCode, string(responseBody))
	}

	// 如果是 204 No Content，直接返回，避免解析空响应体报错
	if response.StatusCode == http.StatusNoContent {
		return res, nil
	}

	// 7. 根据内容类型解码响应体
	switch respContentType {
	case JSON:
		// 使用JSON解码
		if err = json.Decode(response.Body, &res); err != nil {
			return res, errors.WithStack(err)
		}
	case GOB:
		// 使用GOB解码
		if err = gob.Decode(response.Body, &res); err != nil {
			return res, errors.WithStack(err)
		}
	case BYTES:
		// 读取原始字节
		responseBody, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return res, errors.WithStack(readErr)
		}
		// 确保泛型T是[]byte类型
		if v, ok := any(&res).(*[]byte); ok {
			*v = responseBody
		} else {
			return res, errors.Errorf("when contentType is BYTES, T must be []byte, but got %T", res)
		}
	default:
		return res, errors.New("unsupported content type")
	}

	// 8. 返回成功解码的结果
	return res, nil
}
