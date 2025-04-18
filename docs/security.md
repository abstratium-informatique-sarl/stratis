# Security

## Normal Users

Callers should provide a cookie named `token` containing a valid JWT.

This is enforced by using the middleware `SecurityMiddleware`.

If no cookie is provided, the user is anonymous with no roles.

## Service Users

Callers must set the `Authorization` header to contain a valid token which is known to your application.
Provide the `SecurityMiddleware` with a context provider to make this work, 
which is a function that is given a string containing the token, and returns:

- a `jwt.UserContext` object
- a `string` containing the user ID
- a `string` containing the user name
- a `[]string` containing the roles
- an `error` if something went wrong

The context provider must check that the token is valid and that it exists, otherwise your application is liable to BEING INSECURE.

Return the error `fwctx.ErrorTokenNotFound` or `fwctx.ErrorTokenWrong`, both of which cause the framework to abort with a 401 Unauthorized response.

An example could be
that the client passes a string containing the token ID and the actual token with a delimiter between them (so that the token
can be searched for efficiently without actually knowing it, because e.g. only its hash is stored). The context provider
then uses the token ID to look up the token in the database and returns the necessary values used to build the user object. It should also check that the token matches the token ID!

## Summary

In both cases the user which results from either the token cookie or the context provider in combination with the `Authorization` header, should have one of the roles required by the endpoint. If not, the framework will abort with a 403 Forbidden response. If the user is anonymous, the framework will abort with a 401 Unauthorized response.
