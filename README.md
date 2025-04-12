# Stratis

A framework for building microservices in Go, somewhat inspired by Quarkus.

Builds upon [Gin](https://github.com/gin-gonic/gin) and [GORM](https://github.com/go-gorm/gorm).

## Features

- environment variables
- logging
- metrics
- observability
- database (mysql)
- oauth authentication & authorization

## Usage

```go
import "github.com/abstratium-informatique-sarl/stratis/pkg/env"

env.Setup("/path-to-file-containing-secrets.env")
```

## Roadmap

- tracing
- rate limiting
- circuit breaker
- service discovery
- load balancing
- service mesh
- observability

