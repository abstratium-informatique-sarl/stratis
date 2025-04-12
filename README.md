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

```sh
go get github.com/abstratium-informatique-sarl/stratis@v0.0.8

# or

go get github.com/abstratium-informatique-sarl/stratis@latest
```

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

## License

Apache 2.0 => see [LICENSE](LICENSE)

## Authors

Ant Kutschera

## Building / Releasing

```sh
eval "$(ssh-agent -s)"
ssh-add /.../abs.key
export VERS=0.0.x
git add --all && git commit -a -m'<comment>' && git tag v${VERS} && git push origin main v${VERS}
```
