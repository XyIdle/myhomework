# homework2 findMdl的实现

1. 基本上是先匹配 `通配符节点`,然后匹配`正则节点`,然后匹配`参数节点`,最后匹配`普通节点`;
2. 注意nodePathList,需要没遍历一层的时候,都需要替换成对应的childrenList;
3. 对于如果是root.mws要值的, 需要特殊处理,在一开始就做一次apped到最前面;
