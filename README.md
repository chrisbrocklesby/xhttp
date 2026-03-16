# xhttp

Minimal HTTP helper for Go with simple JSON requests, multipart uploads,
and lightweight response handling.

Designed to stay small, readable, and dependency-free.

## Features

-   Simple `GET`, `POST`, `PUT`, `PATCH`, `DELETE`
-   Automatic JSON request encoding
-   Automatic multipart upload when files are present
-   Consistent response parsing
-   Typed HTTP errors
-   Built-in request timeout
-   Lightweight JWT expiry validation

No external dependencies.

------------------------------------------------------------------------

# Install

``` bash
go get github.com/yourname/xhttp
```

------------------------------------------------------------------------

# Import

``` go
import "github.com/yourname/xhttp"
```

------------------------------------------------------------------------

# Types

``` go
type Body map[string]any
type Headers map[string]string
type Response []Body
```

------------------------------------------------------------------------

# Basic Usage

## GET

``` go
res, err := xhttp.Get(
    "https://api.example.com/users",
    xhttp.Headers{
        "Authorization": "Bearer TOKEN",
    },
)

if err != nil {
    panic(err)
}

fmt.Println(res)
```

------------------------------------------------------------------------

## POST JSON

``` go
res, err := xhttp.Post(
    "https://api.example.com/users",
    xhttp.Body{
        "name": "Chris",
        "role": "admin",
    },
    nil,
)
```

------------------------------------------------------------------------

## PUT

``` go
res, err := xhttp.Put(
    "https://api.example.com/users/1",
    xhttp.Body{
        "name": "Updated",
    },
    nil,
)
```

------------------------------------------------------------------------

## PATCH

``` go
res, err := xhttp.Patch(
    "https://api.example.com/users/1",
    xhttp.Body{
        "status": "active",
    },
    nil,
)
```

------------------------------------------------------------------------

## DELETE

``` go
res, err := xhttp.Delete(
    "https://api.example.com/users/1",
    nil,
)
```

------------------------------------------------------------------------

# Headers

``` go
headers := xhttp.Headers{
    "Authorization": "Bearer TOKEN",
    "X-App": "myapp",
}

res, err := xhttp.Get(url, headers)
```

------------------------------------------------------------------------

# File Upload

If a request body contains `*os.File`, the request automatically
switches to `multipart/form-data`.

## Single file

``` go
file, _ := os.Open("photo.jpg")
defer file.Close()

res, err := xhttp.Post(
    "https://api.example.com/upload",
    xhttp.Body{
        "file": file,
        "name": "profile-photo",
    },
    nil,
)
```

## Multiple files

``` go
file1, _ := os.Open("a.jpg")
file2, _ := os.Open("b.jpg")

res, err := xhttp.Post(
    uploadURL,
    xhttp.Body{
        "files": []*os.File{file1, file2},
    },
    nil,
)
```

------------------------------------------------------------------------

# Responses

``` go
type Response []map[string]any
```

Handles both array and single-object JSON responses.

------------------------------------------------------------------------

# Error Handling

``` go
type HTTPError struct {
    StatusCode int
    Status     string
    URL        string
    Body       []byte
}
```

Example:

``` go
res, err := xhttp.Get(url, nil)

if err != nil {
    if httpErr, ok := err.(*xhttp.HTTPError); ok {
        fmt.Println(httpErr.StatusCode)
        fmt.Println(string(httpErr.Body))
    }
}
```

------------------------------------------------------------------------

# JWT Validation

`ValidateJWT` checks:

-   token structure
-   payload decoding
-   `exp` claim
-   expiry time

It **does not verify signatures**.

``` go
err := xhttp.ValidateJWT(token)
if err != nil {
    fmt.Println("invalid token:", err)
}
```

------------------------------------------------------------------------

# Timeout

Default HTTP timeout:

    30 seconds

------------------------------------------------------------------------

# License

MIT
