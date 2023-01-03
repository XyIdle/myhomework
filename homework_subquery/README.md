# homework_subquery

## 场景分析

### SQL语法

1. 作为FROM的table_references

```sql
SELECT * FROM (SELECT * FROM t1 ) AS `sub`;
```

2. 作为SELECT的select_expr

一般用于SELECT字段时，往往将子查询作为别名
 
```sql
SELECT `sub`.`name` FROM (SELECT * FROM t1 ) AS `sub` where `sub`.`id` > 10;
```

3. 作为where的where_condition

```sql
SELECT * FROM t1 WHERE column1 = (SELECT column1 FROM t2);

-- 与ALL、SOME、ANY、IN 等进行搭配
SELECT c1 FROM t1 WHERE c1 > ALL (SELECT c1 FROM t2);
SELECT c1 FROM t1 WHERE c1 <> ANY (SELECT c1 FROM t2);
SELECT c1 FROM t1 WHERE c1 <> SOME (SELECT c1 FROM t2);
SELECT c1 FROM t1 WHERE c1 IN (SELECT c1 FROM t2);
```

4. 作为join的clause

```sql
SELECT `t1`.* FROM (`t1` LEFT JOIN (SELECT * FROM `t2`) AS `sub`) WHERE `t1`.`id` = `sub`.`id`;
```

### 伪代码

1. 作为FROM的table_references

```go
sub := NewSelector[t1](db).AsSubquery("sub")
NewSelector[t1](db).From(sub)
```

2. 作为SELECT的select_expr

```go
sub := NewSelector[t1](db).AsSubquery("sub")
NewSelector[t1](db).Select(sub.C("Id")).From(sub)
```


3. 作为where的where_condition

```go
sub := NewSelector[t1](db).Select(C("Id")).AsSubquery("sub")
NewSelector[t1](db).Where(C("Id").GT(All(sub)
```

4. 作为join的clause

```go
sub := NewSelector[t1](db).AsSubquery("sub")
NewSelector[t2](db).Select(sub.C("Id")).From(t2.Join(sub).On(t2.C("Id").EQ(sub.C("Id"))))
```

### 关于校验

1. 当`select_expr`中包含了不在子查询的结果集字段中时，会返回错误;
2. 当使用`WHERE`、`ON`等条件，对表、子查询的字段做比较时，如果比较的字段类型不同，不做提醒和校验；


## 行业分析

### GORM

GORM在使用子查询时，从测试用例看，并没有特殊的API，或者没有特别的subQuery类型，而是将子查询作为`参数`进行处理。

```go
DB.Select("*").Where("name IN (?)", DB.Select("name").Table("users").Where("name LIKE ?", "subquery_%")).Find(&users).Error;
```

1. `name IN (?)` `?`即为子查询的参数；
2. 子查询`DB.Select("name").Table("users").Where("name LIKE ?", "subquery_%")`，即为参数的值；
