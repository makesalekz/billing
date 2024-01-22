# Ent ORM Files

## Add new Ent Model

```bash
go run -mod=mod entgo.io/ent/cmd/ent new ModelName
```

## Add mixin

1. First, you need `ent/intercept` added to the project. If you have no, add `--feature intercept` to the `ent/generate.go` and run `make ent`.
2. Then copy needed mixin file, fix import paths inside.
3. Then run `make ent`.
