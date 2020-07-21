# Turbolinks

[Turbolinks](https://github.com/turbolinks/turbolinks) middleware for Go.

### Usage

Use it as you would any HTTP middleware.

```go
package main

import (
	"net/http"

	"github.com/bentranter/go-turbolinks"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
    })

	http.ListenAndServe(":3000", turbolinks.Middleware(mux))
}
```

See also the runnable example under the examples directory.

### Dependencies

`turbolinks` depends on `github.com/gorilla/securecookie`.
