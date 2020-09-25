# Locales

## Usage

example for use `zh-CN` language:

```go
// for all Validation.
// NOTICE: must be register before on validate.New()
zhcn.RegisterGlobal()

v := validate.New()

// only for current Validation
zhcn.Register(v)
```
