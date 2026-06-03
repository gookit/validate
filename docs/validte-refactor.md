
重构 validate 库，提示性能和可维护性：

- use pool for Validation?
- cache struct reflect type, tags
- field value struct
- 大部分代码实现放在 internal, 外面通常只定义公共的 type, interface, 包级方法等

现在的痛点：

- 很多地方在重复的使用反射
- 涉及到指针, 结构体 0值 等场景时要么无法判断，要么判断很复杂
  - 主要是往后传递时，丢失了原有类型
- 部分核心逻辑有点混乱了，维护困难
- 没有缓存反射类型等信息，导致性能不高
