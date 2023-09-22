# API Proto files

## Add a proto template

```bash
kratos proto add api/server/v1/server.proto
```

## Generate the proto code

```bash
kratos proto client api/server/v1/server.proto
```

## Generate the source code of service by proto file

```bash
kratos proto server api/server/v1/server.proto -t internal/service
```
