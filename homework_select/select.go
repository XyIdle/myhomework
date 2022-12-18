package orm

import (
	"myhomework/homework_select/internal/errs"
	"myhomework/homework_select/model"
	"strings"
)

// Selector 用于构造 SELECT 语句
type Selector[T any] struct {
	sb      strings.Builder
	db      *DB
	columns []Selectable
	args    []any
	table   string
	model   *model.Model
	where   []Predicate
	having  []Predicate
	orderby []OrderBy
	groupBy []Column
	offset  int
	limit   int
}

func (s *Selector[T]) Select(cols ...Selectable) *Selector[T] {
	s.columns = cols
	return s
}

// From 指定表名，如果是空字符串，那么将会使用默认表名
func (s *Selector[T]) From(tbl string) *Selector[T] {
	s.table = tbl
	return s
}

func (s *Selector[T]) Build() (*Query, error) {
	var (
		t   T
		err error
	)
	s.model, err = s.db.r.Get(&t)
	if err != nil {
		return nil, err
	}
	s.sb.WriteString("SELECT ")
	if err = s.buildColumns(); err != nil {
		return nil, err
	}
	s.sb.WriteString(" FROM ")
	if s.table == "" {
		s.sb.WriteByte('`')
		s.sb.WriteString(s.model.TableName)
		s.sb.WriteByte('`')
	} else {
		s.sb.WriteString(s.table)
	}

	// 构造 WHERE
	if len(s.where) > 0 {
		// 类似这种可有可无的部分，都要在前面加一个空格
		s.sb.WriteString(" WHERE ")
		p := s.where[0]
		for i := 1; i < len(s.where); i++ {
			p = p.And(s.where[i])
		}
		if err = s.buildExpression(p); err != nil {
			return nil, err
		}
	}

	// 构造 GroupBy

	if len(s.groupBy) > 0 {
		s.sb.WriteString(" GROUP BY ")
		for idx, c := range s.groupBy {
			if idx > 0 {
				s.sb.WriteByte(',')
			}
			err := s.buildColumn(c)
			if err != nil {
				return nil, err
			}

		}
	}

	// 构造 Having
	if len(s.having) > 0 {
		s.sb.WriteString(" HAVING ")
		p := s.having[0]
		for i := 1; i < len(s.having); i++ {
			p = p.And(s.having[i])
		}
		if err = s.buildExpression(p); err != nil {
			return nil, err
		}
	}

	// 构造orderby
	if len(s.orderby) > 0 {
		s.sb.WriteString(" ORDER BY ")
		for idx, c := range s.orderby {
			if idx > 0 {
				s.sb.WriteByte(',')
			}
			err := s.buildColumn(Column{name: c.columnName})
			if err != nil {
				return nil, err
			}
			s.sb.WriteString(" " + c.order)
		}
	}

	// 构造 limit
	if s.limit > 0 {
		s.sb.WriteString(" LIMIT ?")
		s.addArg(s.limit)
	}

	// 构造 offset
	if s.offset > 0 {
		s.sb.WriteString(" OFFSET ?")
		s.addArg(s.offset)
	}

	s.sb.WriteString(";")
	return &Query{
		SQL:  s.sb.String(),
		Args: s.args,
	}, nil
}

func (s *Selector[T]) buildExpression(expr Expression) error {
	switch exp := expr.(type) {
	case nil:
	case Predicate:
		// 在这里处理 p
		// p.left 构建好
		// p.op 构建好
		// p.right 构建好
		_, ok := exp.left.(Predicate)
		if ok {
			s.sb.WriteByte('(')
		}
		if err := s.buildExpression(exp.left); err != nil {
			return err
		}
		if ok {
			s.sb.WriteByte(')')
		}

		if exp.op != "" {
			s.sb.WriteByte(' ')
			s.sb.WriteString(exp.op.String())
			s.sb.WriteByte(' ')
		}
		_, ok = exp.right.(Predicate)
		if ok {
			s.sb.WriteByte('(')
		}
		if err := s.buildExpression(exp.right); err != nil {
			return err
		}
		if ok {
			s.sb.WriteByte(')')
		}
	case Column:
		// 这种写法很隐晦
		exp.alias = ""
		return s.buildColumn(exp)
	case value:
		s.sb.WriteByte('?')
		s.addArg(exp.val)
	case RawExpr:
		s.sb.WriteByte('(')
		s.sb.WriteString(exp.raw)
		s.addArg(exp.args...)
		s.sb.WriteByte(')')
	case Aggregate:
		// 聚合函数名
		s.sb.WriteString(exp.fn)
		s.sb.WriteByte('(')
		err := s.buildColumn(Column{name: exp.arg})
		if err != nil {
			return err
		}
		s.sb.WriteByte(')')
		// 聚合函数本身的别名
		if exp.alias != "" {
			s.sb.WriteString(" AS `")
			s.sb.WriteString(exp.alias)
			s.sb.WriteByte('`')
		}
	default:
		return errs.NewErrUnsupportedExpression(expr)
	}
	return nil
}

func (s *Selector[T]) buildColumns() error {
	if len(s.columns) == 0 {
		// 没有指定列
		s.sb.WriteByte('*')
		return nil
	}

	for i, col := range s.columns {
		if i > 0 {
			s.sb.WriteByte(',')
		}
		switch c := col.(type) {
		case Column:
			err := s.buildColumn(c)
			if err != nil {
				return err
			}
		case Aggregate:
			// 聚合函数名
			s.sb.WriteString(c.fn)
			s.sb.WriteByte('(')
			err := s.buildColumn(Column{name: c.arg})
			if err != nil {
				return err
			}
			s.sb.WriteByte(')')
			// 聚合函数本身的别名
			if c.alias != "" {
				s.sb.WriteString(" AS `")
				s.sb.WriteString(c.alias)
				s.sb.WriteByte('`')
			}
		case RawExpr:
			s.sb.WriteString(c.raw)
			s.addArg(c.args...)
		}
	}

	return nil
}

func (s *Selector[T]) buildColumn(c Column) error {
	s.sb.WriteByte('`')
	fd, ok := s.model.FieldMap[c.name]
	if !ok {
		return errs.NewErrUnknownField(c.name)
	}
	s.sb.WriteString(fd.ColName)
	s.sb.WriteByte('`')
	if c.alias != "" {
		s.sb.WriteString(" AS `")
		s.sb.WriteString(c.alias)
		s.sb.WriteByte('`')
	}
	return nil
}

func (s *Selector[T]) addArg(vals ...any) {
	if len(vals) == 0 {
		return
	}
	if s.args == nil {
		s.args = make([]any, 0, 8)
	}
	s.args = append(s.args, vals...)
}

// Where 用于构造 WHERE 查询条件。如果 ps 长度为 0，那么不会构造 WHERE 部分
func (s *Selector[T]) Where(ps ...Predicate) *Selector[T] {
	s.where = ps
	return s
}

// GroupBy 设置 group by 子句
func (s *Selector[T]) GroupBy(cols ...Column) *Selector[T] {
	s.groupBy = cols
	return s
}

func (s *Selector[T]) Having(ps ...Predicate) *Selector[T] {
	s.having = ps
	return s
}

func (s *Selector[T]) Offset(offset int) *Selector[T] {
	s.offset = offset
	return s
}

func (s *Selector[T]) Limit(limit int) *Selector[T] {
	s.limit = limit
	return s
}

func (s *Selector[T]) OrderBy(orderBys ...OrderBy) *Selector[T] {
	s.orderby = orderBys
	return s
}

func NewSelector[T any](db *DB) *Selector[T] {
	return &Selector[T]{
		db: db,
	}
}

type Selectable interface {
	selectable()
}

type OrderBy struct {
	columnName string
	order      string
}

func Asc(col string) OrderBy {

	return OrderBy{
		columnName: col,
		order:      "ASC",
	}
}

func Desc(col string) OrderBy {
	return OrderBy{
		columnName: col,
		order:      "DESC",
	}
}
