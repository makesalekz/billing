# dummy

## Init project

1. Change module name in go.mod
2. Rename cmd/dummy directory
3. Replace every "dummy" code with your code
4. Local env: create .env from .example, replace SERVICE_NAME, set unused HTTP_PORT, GRPC_PORT
5. CI: create .gitlab-ci.yml from .example, replace "DB_PORT: 5440X" with unused port

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

### Download and update dependencies

```bash
make init
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
