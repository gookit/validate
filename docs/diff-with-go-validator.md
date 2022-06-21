# Diff with go validator

How is it different compared to https://github.com/go-playground/validator

- Most filters and validators have aliases for convenience
- Support filter/sanitize/convert data before validate
- Simple and fast configuration and validation of `Map` data
- Support scene settings, verify different fields in different scenes
- Quickly validate `http.Request` and collect data based on the request data type `Content-Type`
- Supports direct use of rules to validate value. eg: `validate.Val("xyz@mail.com", "required|email")`
- ...etc.

- 大多数过滤器和验证器都有别名方便使用
- 能根据请求数据类型 `Content-Type` 快速验证 `http.Request` 并收集数据
- 能简单快速的配置规则并验证 Map 数据