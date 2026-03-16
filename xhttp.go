package xhttp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Body map[string]any
type Form map[string]string
type Headers map[string]string

// Response wraps any JSON value returned by an HTTP call.
// Use the accessor methods instead of raw type assertions.
type Response struct {
	data any
}

// Key returns the value for key if the response holds a JSON object.
func (r Response) Key(key string) Response {
	if m, ok := r.data.(map[string]any); ok {
		return Response{data: m[key]}
	}
	return Response{}
}

// Array returns the items as []Response if the response holds a JSON array.
func (r Response) Array() []Response {
	s, ok := r.data.([]any)
	if !ok {
		return nil
	}
	out := make([]Response, len(s))
	for i, v := range s {
		out[i] = Response{data: v}
	}
	return out
}

// String returns the string value, or "" if the value is not a string.
func (r Response) String() string {
	s, _ := r.data.(string)
	return s
}

// Float returns the numeric value, or 0 if the value is not a number.
func (r Response) Float() float64 {
	f, _ := r.data.(float64)
	return f
}

// Int returns the integer value, or 0 if the value is not a number.
func (r Response) Int() int {
	f, _ := r.data.(float64)
	return int(f)
}

// Bool returns the boolean value, or false if the value is not a bool.
func (r Response) Bool() bool {
	b, _ := r.data.(bool)
	return b
}

// Raw returns the underlying value for cases that need direct access.
func (r Response) Raw() any {
	return r.data
}

const defaultTimeout = 30 * time.Second

var defaultClient = &http.Client{
	Timeout: defaultTimeout,
}

type HTTPError struct {
	StatusCode int
	Status     string
	URL        string
	Body       []byte
}

func (e *HTTPError) Error() string {
	if len(e.Body) == 0 {
		return fmt.Sprintf("request failed: %s %s", e.Status, e.URL)
	}
	return fmt.Sprintf("request failed: %s %s: %s", e.Status, e.URL, string(e.Body))
}

func Post(url string, payload any, headers Headers) (Response, error) {
	return request(http.MethodPost, url, payload, headers)
}

func Get(url string, headers Headers) (Response, error) {
	return request(http.MethodGet, url, nil, headers)
}

func Delete(url string, headers Headers) (Response, error) {
	return request(http.MethodDelete, url, nil, headers)
}

func Patch(url string, payload any, headers Headers) (Response, error) {
	return request(http.MethodPatch, url, payload, headers)
}

func Put(url string, payload any, headers Headers) (Response, error) {
	return request(http.MethodPut, url, payload, headers)
}

func request(method, url string, payload any, headers Headers) (Response, error) {
	reqBody, contentType, err := encodePayload(payload)
	if err != nil {
		return Response{}, err
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return Response{}, fmt.Errorf("create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	applyHeaders(req.Header, headers)

	resp, err := defaultClient.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return Response{}, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			URL:        url,
			Body:       respBytes,
		}
	}

	return decodeResponse(respBytes)
}
func encodePayload(payload any) (io.Reader, string, error) {
	if payload == nil {
		return nil, "", nil
	}

	switch v := payload.(type) {
	case Body:
		if hasFiles(v) {
			return buildMultipart(v)
		}
		b, err := json.Marshal(v)
		if err != nil {
			return nil, "", fmt.Errorf("marshal json body: %w", err)
		}
		return bytes.NewReader(b), "application/json", nil
	case Form:
		return encodeForm(v), "application/x-www-form-urlencoded", nil
	default:
		return nil, "", fmt.Errorf("unsupported payload type %T (expected xhttp.Body, xhttp.Form, or nil)", payload)
	}
}

func encodeForm(form Form) io.Reader {
	values := url.Values{}
	for k, v := range form {
		values.Set(k, v)
	}

	return strings.NewReader(values.Encode())
}

func applyHeaders(dst http.Header, headers Headers) {
	for k, v := range headers {
		dst.Set(k, v)
	}
}

func hasFiles(body Body) bool {
	for _, v := range body {
		switch v.(type) {
		case *os.File, []*os.File:
			return true
		}
	}
	return false
}

func buildMultipart(body Body) (io.Reader, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for key, value := range body {
		switch v := value.(type) {
		case *os.File:
			if err := writeFilePart(writer, key, v); err != nil {
				return nil, "", err
			}
		case []*os.File:
			for _, f := range v {
				if err := writeFilePart(writer, key, f); err != nil {
					return nil, "", err
				}
			}
		default:
			if err := writeField(writer, key, value); err != nil {
				return nil, "", err
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

func writeFilePart(writer *multipart.Writer, field string, file *os.File) error {
	if file == nil {
		return fmt.Errorf("file field %q is nil", field)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind file %q: %w", field, err)
	}

	part, err := writer.CreateFormFile(field, filepath.Base(file.Name()))
	if err != nil {
		return fmt.Errorf("create form file %q: %w", field, err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copy file %q: %w", field, err)
	}

	return nil
}

func writeField(writer *multipart.Writer, field string, value any) error {
	switch v := value.(type) {
	case string:
		if err := writer.WriteField(field, v); err != nil {
			return fmt.Errorf("write field %q: %w", field, err)
		}
		return nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshal field %q: %w", field, err)
		}
		if err := writer.WriteField(field, string(b)); err != nil {
			return fmt.Errorf("write field %q: %w", field, err)
		}
		return nil
	}
}

func decodeResponse(data []byte) (Response, error) {
	if len(data) == 0 {
		return Response{}, nil
	}

	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return Response{}, fmt.Errorf("decode response: %w", err)
	}

	return Response{data: result}, nil
}

// ValidateJWT only checks JWT shape and exp claim.
// It does not verify the signature.
func ValidateJWT(token string) error {
	parts := bytes.Split([]byte(token), []byte("."))
	if len(parts) != 3 {
		return errors.New("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(string(parts[1]))
	if err != nil {
		return fmt.Errorf("decode token payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fmt.Errorf("decode token claims: %w", err)
	}

	exp, ok := claims["exp"]
	if !ok {
		return errors.New("token missing exp claim")
	}

	expFloat, ok := exp.(float64)
	if !ok {
		return errors.New("token exp claim is not numeric")
	}

	if time.Unix(int64(expFloat), 0).Before(time.Now()) {
		return errors.New("token has expired")
	}

	return nil
}
