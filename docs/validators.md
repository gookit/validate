# validators

- `notIn`
- `unique/distinct` For arrays & slices, unique will ensure that there are no duplicates. For maps, unique will ensure that there are no duplicate values.

- dateFormat	验证是否是 date, 并且是指定的格式	['publishedAt', 'dateFormat', 'Y-m-d']
- dateEquals	验证是否是 date, 并且是否是等于给定日期	['publishedAt', 'dateEquals', '2017-05-12']
- beforeDate	验证字段值必须是给定日期之前的值(ref laravel)	['publishedAt', 'beforeDate', '2017-05-12']
- beforeOrEqualDate	字段值必须是小于或等于给定日期的值(ref laravel)	['publishedAt', 'beforeOrEqualDate', '2017-05-12']
- afterOrEqualDate	字段值必须是大于或等于给定日期的值(ref laravel)	['publishedAt', 'afterOrEqualDate', '2017-05-12']
- afterDate