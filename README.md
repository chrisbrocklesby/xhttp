# xhttp

Minimal HTTP helper package for Go. Supports JSON, form-encoded payloads, and file uploads
out of the box — with fluent response accessors and no external dependencies.

Designed to stay small, readable, and dependency-free.

## Features

-   Simple `GET`, `POST`, `PUT`, `PATCH`, `DELETE`
-   Automatic JSON or multipart encoding (detects files automatically)
-   Supports `application/x-www-form-urlencoded` via `xhttp.Form`
-   Fluent response accessors — no raw type assertions
-   Typed HTTP errors
-   Built-in request timeout
-   Lightweight JWT expiry validation

No external dependencies.

------------------------------------------------------------------------

# Install

``` bash
go get github.com/chrisbrocklesby/xhttp
```

------------------------------------------------------------------------

# Import

``` go
import "github.com/chrisbrocklesby/xhttp"
```

------------------------------------------------------------------------

# Types

``` go
type Body map[string]any
type Form map[string]string
type Headers map[string]string
type Response struct{ ... }
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

## POST Form URL Encoded

``` go
res, err := xhttp.Post(
    "https://api.example.com/login",
    xhttp.Form{
        "username": "chris",
        "password": "secret",
    },
    nil,
)
```

Use the same pattern with `xhttp.Put(...)` and `xhttp.Patch(...)`.

## Payload Encoding Rules

For `xhttp.Post(...)`, `xhttp.Put(...)`, and `xhttp.Patch(...)`, encoding is
chosen from the payload type:

| Payload value | Content-Type | Behavior |
|---|---|---|
| `xhttp.Body{...}` without files | `application/json` | JSON body |
| `xhttp.Body{...}` with `*os.File` / `[]*os.File` | `multipart/form-data` | File upload + form parts |
| `xhttp.Form{...}` | `application/x-www-form-urlencoded` | URL-encoded form body |
| `nil` | none | No request body |

If you pass any other payload type, the request returns an error.

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

`Response` wraps any JSON value (object, array, string, number, bool, or null)
and provides fluent accessor methods so you never need raw type assertions.

## Methods

| Method | Returns | Description |
|---|---|---|
| `.Key("key")` | `Response` | Navigate into a JSON object by key |
| `.Array()` | `[]Response` | Iterate a JSON array |
| `.String()` | `string` | Extract a string value (or `""`) |
| `.Int()` | `int` | Extract an integer value (or `0`) |
| `.Float()` | `float64` | Extract a float value (or `0`) |
| `.Bool()` | `bool` | Extract a boolean value (or `false`) |
| `.Raw()` | `any` | Access the underlying value directly |

All methods are safe to chain — calling `.Key` on a non-object or `.String` on
a non-string returns the zero value rather than panicking.

## Accessing an object field

``` go
res, err := xhttp.Post("https://api.example.com/auth", xhttp.Body{
    "identity": "user@example.com",
    "password": "secret",
}, nil)

token := res.Key("token").String()
```

## Iterating an array

``` go
res, err := xhttp.Get("https://api.example.com/posts", nil)

for _, post := range res.Key("items").Array() {
    fmt.Println(post.Key("title").String())
}
```

## Nested access

``` go
city := res.Key("address").Key("city").String()
```

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

# JWT Validation (Expiry Checker)

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
