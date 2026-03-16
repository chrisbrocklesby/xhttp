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
	"os"
	"path/filepath"
	"time"
)

type Body map[string]any
type Headers map[string]string
type Response []Body

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

func Post(url string, body Body, headers Headers) (Response, error) {
	return request(http.MethodPost, url, body, headers)
}

func Get(url string, headers Headers) (Response, error) {
	return request(http.MethodGet, url, nil, headers)
}

func Delete(url string, headers Headers) (Response, error) {
	return request(http.MethodDelete, url, nil, headers)
}

func Patch(url string, body Body, headers Headers) (Response, error) {
	return request(http.MethodPatch, url, body, headers)
}

func Put(url string, body Body, headers Headers) (Response, error) {
	return request(http.MethodPut, url, body, headers)
}

func request(method, url string, body Body, headers Headers) (Response, error) {
	reqBody, contentType, err := encodeBody(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	applyHeaders(req.Header, headers)

	resp, err := defaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			URL:        url,
			Body:       respBytes,
		}
	}

	return decodeResponse(respBytes)
}

func encodeBody(body Body) (io.Reader, string, error) {
	if body == nil {
		return nil, "", nil
	}

	if hasFiles(body) {
		return buildMultipart(body)
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, "", fmt.Errorf("marshal json body: %w", err)
	}

	return bytes.NewReader(b), "application/json", nil
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

	var list []Body
	if err := json.Unmarshal(data, &list); err == nil {
		return list, nil
	}

	var item Body
	if err := json.Unmarshal(data, &item); err == nil {
		return Response{item}, nil
	}

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return Response{}, nil
	}

	return nil, errors.New("response is not a JSON object or array of objects")
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
