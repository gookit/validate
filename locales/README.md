# Locales

## Usage

example for use `zh-CN` language:

```go
// for all Validation.
// NOTICE: must call before on 
zhcn.RegisterGlobal()

v := validate.New()

// only for current Validation
zhcn.Register(v)
```
