package plugins

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/evilsocket/islazy/log"
)

type httpPackage struct {
}

var hp = httpPackage{}

func GetHTTP() httpPackage {
	return hp
}

type httpResponse struct {
	Error    error
	Response *http.Response
	Raw      []byte
	Body     string
}

func (c *httpPackage) createRequest(method string, uri string, headers map[string]string, data interface{}) (*http.Request, error) {
	var reader io.Reader

	if data != nil {
		switch t := data.(type) {
		case string:
			reader = bytes.NewBufferString(t)

		case map[string]string:
			dt := url.Values{}
			for k, v := range t {
				dt.Set(k, v)
			}
			reader = bytes.NewBufferString(dt.Encode())

		case *bytes.Buffer:
			reader = t

		default:
			return nil, fmt.Errorf("wrong data type")
		}
	}

	req, err := http.NewRequest(method, uri, reader)
	if err != nil {
		return nil, err
	}

	for name, value := range headers {
		req.Header.Add(name, value)
	}

	return req, nil
}

func (c *httpPackage) Request(method string, uri string, headers map[string]string, data interface{}) httpResponse {
	client := &http.Client{}

	req, err := c.createRequest(method, uri, headers, data)
	if err != nil {
		log.Error("http.createRequest : %s", err)
		return httpResponse{Error: err}
	}

	resp, err := client.Do(req)
	if err != nil {
		return httpResponse{Error: err}
	}
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return httpResponse{Error: err}
	}

	return httpResponse{
		Error:    nil,
		Response: resp,
		Raw:      raw,
		Body:     string(raw),
	}
}

func (c *httpPackage) Get(url string, headers map[string]string) httpResponse {
	return c.Request("GET", url, headers, nil)
}

func (c *httpPackage) Post(url string, headers map[string]string, data interface{}) httpResponse {
	return c.Request("POST", url, headers, data)
}

func (c *httpPackage) DownloadFile(filepath string, method string, uri string, headers map[string]string, data interface{}) httpResponse {
	client := &http.Client{}

	req, err := c.createRequest(method, uri, headers, data)
	if err != nil {
		log.Error("http.createRequest : %s", err)
		return httpResponse{Error: err}
	}

	resp, err := client.Do(req)
	if err != nil {
		return httpResponse{Error: err}
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		log.Error("http.DownloadFile: %s", err)
		return httpResponse{Error: err}
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		log.Error("%s", err)
		return httpResponse{Error: err}
	}

	return httpResponse{}
}

func (c *httpPackage) UploadFile(method string, uri string, headers map[string]string, data interface{}, filename string, fieldname string) httpResponse {
	client := &http.Client{}

	file, err := os.Open(filename)
	if err != nil {
		log.Error("%s", err)
		return httpResponse{Error: err}
	}
	defer file.Close()

	bodyfile := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyfile)
	part, err := writer.CreateFormFile(fieldname, filepath.Base(filename))

	if v, ok := data.(map[string]string); ok {
		for key, val := range v {
			_ = writer.WriteField(key, val)
		}
	}

	if err != nil {
		log.Error("%s", err)
		return httpResponse{Error: err}
	}

	io.Copy(part, file)
	writer.Close()

	req, err := c.createRequest(method, uri, headers, bodyfile)
	if err != nil {
		log.Error("http.createRequest : %s", err)
		return httpResponse{Error: err}
	}

	resp, err := client.Do(req)
	if err != nil {
		return httpResponse{Error: err}
	}
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return httpResponse{Error: err}
	}

	return httpResponse{
		Error:    nil,
		Response: resp,
		Raw:      raw,
		Body:     string(raw),
	}
}