/*
Package dihttp provides HTTP middleware for creating [di.Container] scopes for each request.

Example:

	package main

	import (
		"net/http"

		"github.com/johnrutherford/di-kit"
		"github.com/johnrutherford/di-kit/dihttp"
	)

	func main() {
		c, err := di.NewContainer(
			di.WithService(NewService),
			di.WithService(NewOtherService, di.Scoped),
		)

		// Create a new scope middleware
		scopeMiddleware := dihttp.NewScopeMiddleware(c)

		// Create a handler function
		handler := func(w http.ResponseWriter, r *http.Request) {
			svc := dicontext.MustResolve[OtherService](r.Context())

			svc.HandleRequest(w, r)
		}

		// Wrap the handler with the scope middleware
		wrappedHandler := scopeMiddleware(handler)

		// Start the HTTP server
		http.HandleFunc("/", wrappedHandler)
		fmt.Println("Server started on port 8080")
		http.ListenAndServe(":8080", nil)
	}
*/
package dihttp
