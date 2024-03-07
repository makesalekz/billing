# dummy

## Init project

1. Change module name in go.mod
2. Replace every "dummy" code with your code
3. Local env: replace SERVICE_NAME, set unused HTTP_PORT, GRPC_PORT in .env.example, then copy .env.example into .env
4. CI: create .gitlab-ci.yml from .example, replace "DB_PORT: 5440X" with unused port

### 1. Download and update dependencies

```bash
make init
```

### 2. Create proto for service

```bash
kratos proto add api/{project-name}/v1/{service_name}.proto
```

Change package name to `{project_name}.v1` in this new proto (and in `errors.proto` as well) and fix package name below (use dash and underscore wisely):

```proto
option go_package = "gitlab.calendaria.team/services/{project-name}/api/{project-name}/v1;{project_name}_v1";
option java_multiple_files = true;
option java_package = "{project_name}.v1";
```

### 3. Download and update go modules

```bash
go mod tidy
```

### 4. Generate API pb files

```bash
make api
```

### 5. Create service

```bash
kratos proto server api/{project-name}/v1/{service_name}.proto -t internal/service
```

### 6. Add service to `internal/server/http.go` and/or `internal/server/grpc.go`

6.1. Check commented strings in these files and edit.
6.2. Add service to `internal/service/service.go`

### 7. Create your first ent schema

```bash
go run -mod=mod entgo.io/ent/cmd/ent new {Model}
```

Create at least one field in the created schema, in order to fully prepare the project.

### 8. Generate ent files

1. Revert back auto-fixed `ent/generate.go`
2. Generate ent files:

```bash
make ent
```

### 9. Create your first repo in `internal/data/{entities}.go`

```go
package data

import (
    "context"

    "gitlab.calendaria.team/services/{project-name}/ent"

    _ "github.com/lib/pq"
)

// EntityRepo
type EntityRepo interface {
    CreateEntity(ctx context.Context, string) (*ent.Entity, error)
}

type entitiesRepo struct {
    db *ent.Client
}

// NewEntityRepo .
func NewEntityRepo(d *Data) EntityRepo {
    return &entitiesRepo{
        db: d.db,
    }
}

func (r *entitiesRepo) CreateEntity(ctx context.Context, some string) (*ent.Entity, error) {
    return r.db.Entity.Create().
        SetSome(some).
        Save(ctx)
}
```

And add it to `internal/data/data.go`:

```go
var ProviderSet = wire.NewSet(
    // ...
    NewEntityRepo,
)
```

### 10. Uncomment import line in `internal/data/data.go` in order to avoid ent cycle import

```bash
  _ "gitlab.calendaria.team/services/{project-name}/ent/runtime"
```

### 11. Create your first use-case in `internal/biz/{use_case}.go`

```go
package biz

import (
    "context"

    v1 "gitlab.calendaria.team/services/{project-name}/api/{project-name}/v1"
)

// SomeUsecase .
type SomeUsecase struct {
    repo data.EntityRepo
}

// NewGreeterUsecase .
func NewTranscriptionsUsecase(
    repo data.EntityRepo,
) (*SomeUsecase, error) {
    return &SomeUsecase{
        repo: repo,
    }, nil
}

func (uc *SomeUsecase) CreateEntity(ctx context.Context, some string) (*ent.Entity, error) {
    return uc.repo.CreateEntity(ctx, some)
}
```

And add it to `internal/biz/biz.go`:

```go
var ProviderSet = wire.NewSet(
    // ...
    SomeUsecase,
)
```

### 12. Add use-case to service `internal/service/{service_name}.go`

```go
type ServiceNameService struct {
    // ...

    uc *biz.SomeUsecase
}

func NewServiceNameService(uc *biz.SomeUsecase) *ServiceNameService {
    return &ServiceNameService{
        uc: uc,
    }
}
```

### 13. Generate wire

```bash
make generate
```

You are finally set! Remove this section from `README.md` and make an initial commit.

## Proto files

### Add a proto template

```bash
kratos proto add api/server/server.proto
```

### Generate the proto code

```bash
kratos proto client api/server/server.proto
```

### Generate the source code of service by proto file

```bash
kratos proto server api/server/server.proto -t internal/service

go generate ./...
```

## Generate other auxiliary files by Makefile

### Download and update dependencies if needed

```bash
make init
```

### Copy proto files from vendor to third_party/api

```bash
make proto
```

### Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file

```bash
make api
```

### Generate all files

```bash
make all
```

### Generate migrations

[Install Atlas](https://entgo.io/docs/versioned-migrations#generating-migrations)

```bash
make migrations
```

## Run

### Run debug

```bash
make run
```

### Build & Run

```bash
export AWS_ACCESS_KEY_ID={aws-key}
export AWS_SECRET_ACCESS_KEY={aws-secret}

make build
```

## Run in Docker

```bash
make start
```

To stop docker:

```bash
make stop
```
