# Security

Callers should provide a cookie named `token` containing a valid JWT.

This is checked by using the middleware `SecurityMiddleware`.

Or they can set the `Authorization` header to container a valid token which has been provided by the application,
using the context provider. Use the `ServiceUserSecurityMiddleware` for this, as it is designed for service users.
You must provide a context provider, 
which is a function that is given a string containing the token, and returns:

- a `jwt.UserContext` object
- a `string` containing the user ID
- a `string` containing the user name
- a `[]string` containing the roles
- an `error` if something went wrong

An example could be
that the client passes a string containing the token ID and the actual token. The context provider
then uses the token ID to look up the token in the database and returns the necessary values used to build the user object. It should also check that the token matches the token ID!

In both cases the user which results from either the token cookie or the context provider in combination with the `Authorization` header, should have one of the roles required by the endpoint.