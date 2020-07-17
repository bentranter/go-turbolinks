# Turbolinks

Turbolinks middleware for Go.

### Known Issues

* The internal redirect doesn't correctly set the magic Turbolinks header such that the final URL is updated. For example, if you click a link to a handler that redirects to a normal page, the URL will set to the redirect handler instead of the final destination.

### Usage

Use it as middleware.

### Dependencies

`turbolinks` depends on `github.com/gorilla/securecookie`.

### Testing
