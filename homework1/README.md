# homework1 代码梳理



> 由于homework1在gitee中已经把答案暴露出来了，因此针对代码逻辑做了梳理说明



## 一. 关于通配符匹配



### 1.1. findRoute理解

核心逻辑依然是 `findRoute`

```go
func (r *router) findRoute(method string, path string) (*matchInfo, bool) {
	root, ok := r.trees[method]
	if !ok {
		return nil, false
	}

	if path == "/" {
		return &matchInfo{n: root}, true
	}

	segs := strings.Split(strings.Trim(path, "/"), "/")
	mi := &matchInfo{}
	for _, s := range segs {
		var child *node
		child, ok = root.childOf(s)   // 1. 尝试找子节点
		if !ok {
      if root.typ == nodeTypeAny { // 3. 如果当前seg 没有找到，先看一下root(current)的类型是不是”通配符“
				mi.n = root                //   3.1. 如果是通配符类型，则把current赋值给当前matchInfo的node，然后返回
				return mi, true
			}
			return nil, false
		}
		if child.paramName != "" {        // 4. 如果当前有参数，则将seg的值，作为参数的value添加到pathParam
			mi.addValue(child.paramName, s)
		}
		root = child                // 2. 如果普通节点，就把子节点赋值给root，这里的root其实充当了current指针
	}
	mi.n = root
	return mi, true
}
```



### 1.2. childOf理解

寻找路由的过程，本质上就是遍历树的过程。因此代码中为了确定每个节点的类型，扩展了`node`,增加了`typ`字段，以区分是通配符node（`nodeTypeAny`）还是正则node(`nodeTypeReg`)

同时，为了代码的可读性和逻辑性，对`childOf`增加了helper函数`childOfNonStatic`

```go
// 以 /a/b/c 匹配 /a/b/* 为例
// path传入的顺序为 ["a","b","c"]
func (n *node) childOf(path string) (*node, bool) {
	if n.children == nil {                  // 2. b没有普通children，因此这里为true，则进一步交给childOfNonStatic判断
		return n.childOfNonStatic(path)
	}
	res, ok := n.children[path]          // 1. a存在子节点b，因此返回 b节点
	if !ok {
		return n.childOfNonStatic(path)
	}
	return res, ok
}


func (n *node) childOfNonStatic(path string) (*node, bool) {
	if n.regChild != nil {   //  3. b下面没有 正则节点
		if n.regChild.regExpr.Match([]byte(path)) {
			return n.regChild, true
		}
	}
	if n.paramChild != nil {  //  4. b下面没有 参数节点
		return n.paramChild, true
	}
	return n.starChild, n.starChild != nil  // 5. 直接返回*节点，以及是否实现handler
}
```



#### 1.2.1. /a/b/* 路由定义

```go
http.MethodGet: {
				path: "/",
				children: map[string]*node{	
        // 省略其他
					"a": {
						path: "a",
						children: map[string]*node{ // 2. b是a的子节点， 且b也是静态匹配
							"b": {
								path: "b",   // 3. b没有普通子节点，只有一个 *节点
								starChild: &node{
									path:    "*",
									handler: mockHandler,
									typ:     nodeTypeAny,
								},
								handler: mockHandler,
								typ:     nodeTypeStatic,
							},
						},
						typ: nodeTypeStatic,   // 1. 首先 a 是静态匹配
					},
           // 省略其他
          starChild: {/*省略其他*/},
          handler: mockHandler,
					typ:     nodeTypeStatic,
```



`GET /a/b/c/d/e`的匹配过程：

1. path会被切割成`["a", "b", "c", "d", "e"]`, 然后对元素循环调用`childOf`方法
2. 首先找到`GET`方法的tree，其`root`节点为`/`;
3. `a` 是`root`的children，且类型为`nodeTypeStatic`;
   - 此时的root（current）指向了`a`
4. `b`是`a`的children，且类型为`nodeTypeStatic`;
   - 此时的root（current）指向了`b`
5. 当遍历到`c`时，由于`b`没有`children`， 走`childOfNonStatic`函数 ，然后此时的`(n *node)`还是上一次循环中的`b`,因此当返回`n.starChild`的时候，就是返回的`通配符`节点（`path=*`）
6. 后续的`d, e...`匹配过程，和c一样，始终走`childOfNonStatic`函数，返回的永远是`b.starChild`

至此就可以无限匹配下去。

-----

## 二. 正则匹配

为了引入正则匹配，需要对`node`节点进行扩展：

```go
// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 正则匹配，形式 :param_name(reg_expr)
// 3. 路径参数匹配：形式 :param_name
// 4. 通配符匹配：*
// 这是不回溯匹配
type node struct {
    typ nodeType

    path string
    // children 子节点
    // 子节点的 path => node
    children map[string]*node
    // handler 命中路由之后执行的逻辑
    handler HandleFunc

    // 通配符 * 表达的节点，任意匹配
    starChild *node

    paramChild *node
    // 正则路由和参数路由都会使用这个字段
    paramName string

    // 正则表达式
    regChild *node             // 1. 为了引入正则节点，新增的字段
    regExpr  *regexp.Regexp    // 2. 提取路由中的正则表达式，并在创建节点childOrCreateReg的时候事先编译好，放入该字段
}
```



### 2.1. 创建正则节点

`addRoute`调用 `childOrCreateReg`，创建相应节点

#### 2.1.1. childOrCreate

```go
func (n *node) childOrCreate(path string) *node {
    if path == "*" {
        if n.paramChild != nil {
            panic(fmt.Sprintf("web: 非法路由，已有路径参数路由。不允许同时注册通配符路由和参数路由 [%s]", path))
        }
        if n.regChild != nil {
            panic(fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册通配符路由和正则路由 [%s]", path))
        }
        if n.starChild == nil {
            n.starChild = &node{path: path, typ: nodeTypeAny}
        }
        return n.starChild
    }

    // 以 : 开头，需要进一步解析，判断是参数路由还是正则路由
    if path[0] == ':' {
        paramName, expr, isReg := n.parseParam(path)  // 1. 对path路径进行 参数 和 是否正则 解析
        if isReg {  
            return n.childOrCreateReg(path, expr, paramName) // 2. 如果是正则，则创建正则节点，这里兼顾了参数名字的传递
        }
        return n.childOrCreateParam(path, paramName) // 3. 如果不是正则，则视为 参数节点
    }

   // 4. 最终会找到/创建普通节点
    if n.children == nil {
        n.children = make(map[string]*node)
    }
    child, ok := n.children[path]
    if !ok {
        child = &node{path: path, typ: nodeTypeStatic}
        n.children[path] = child
    }
    return child
}

func (n *node) parseParam(path string) (string, string, bool) {
    // 去除 :
    path = path[1:]
    segs := strings.SplitN(path, "(", 2)
    if len(segs) == 2 {
        expr := segs[1]
        if strings.HasSuffix(expr, ")") {
            return segs[0], expr[:len(expr)-1], true
        }
    }
    return path, "", false
}
```



#### 2.1.2. childOrCreateReg

```go
func (n *node) childOrCreateReg(path string, expr string, paramName string) *node {
    if n.starChild != nil {
        panic(fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和正则路由 [%s]", path))
    }
    if n.paramChild != nil {
        panic(fmt.Sprintf("web: 非法路由，已有路径参数路由。不允许同时注册正则路由和参数路由 [%s]", path))
    }
    if n.regChild != nil { 
        if n.regChild.regExpr.String() != expr || n.paramName != paramName {
            panic(fmt.Sprintf("web: 路由冲突，正则路由冲突，已有 %s，新注册 %s", n.regChild.path, path))
        }
    } else {  // 1. 经过上面的校验，说明当前路径是可以添加的正则节点
        regExpr, err := regexp.Compile(expr)  // 2. 对parseParam提取的正则表达式，进行预编译
        if err != nil {
            panic(fmt.Errorf("web: 正则表达式错误 %w", err))
        }
        // 3. 创建一个带有正则表达式的节点
        //  正则表达式的值，可以使用 n.paramName 获取
        n.regChild = &node{path: path, paramName: paramName, regExpr: regExpr, typ: nodeTypeReg}
    }
    return n.regChild  // 4. 返回该节点
}
```



### 2.2. 查找正则节点



```go
// 还是这个函数
// 优先级： 正则检查 > 纯参数 > 通配符
func (n *node) childOfNonStatic(path string) (*node, bool) {
	if n.regChild != nil {   //  1. regChild不为空
		if n.regChild.regExpr.Match([]byte(path)) {  // 2. 通过regExpr检查path是否满足正则表达式
			return n.regChild, true                    // 3. 如果满足，则返回该正则节点
		}
	}
	if n.paramChild != nil { 
		return n.paramChild, true
	}
	return n.starChild, n.starChild != nil 
}
```



