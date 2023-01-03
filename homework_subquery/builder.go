package orm

import (
	"myhomework/homework_subquery/internal/errs"
	"myhomework/homework_subquery/model"
	"strings"
)

type builder struct {
	core
	sb      strings.Builder
	args    []any
	dialect Dialect
	quoter  byte
	model   *model.Model
}

// buildColumn 构造列
// 如果 table 没有指定，我们就用 model 来判断列是否存在
func (b *builder) buildColumn(table TableReference, fd string) error {
	var alias string
	if table != nil {
		alias = table.tableAlias()
	}
	if alias != "" {
		b.quote(alias)
		b.sb.WriteByte('.')
	}
	colName, err := b.colName(table, fd)
	if err != nil {
		return err
	}
	b.quote(colName)
	return nil
}

func (b *builder) colName(table TableReference, fd string) (string, error) {
	switch tab := table.(type) {
	case nil:
		fdMeta, ok := b.model.FieldMap[fd]
		if !ok {
			return "", errs.NewErrUnknownField(fd)
		}
		return fdMeta.ColName, nil
	case Table:
		m, err := b.r.Get(tab.entity)
		if err != nil {
			return "", err
		}
		fdMeta, ok := m.FieldMap[fd]
		if !ok {
			return "", errs.NewErrUnknownField(fd)
		}
		return fdMeta.ColName, nil
	case Subquery:
		foundCol := false
		if len(tab.column) > 0 {
			for _, v := range tab.column {
				if fd == v.fieldName() {
					foundCol = true
					break
				}
			}

			if !foundCol {
				return "", errs.NewErrUnknownField(fd)
			}

		}

		fdMeta, ok := tab.model.FieldMap[fd]

		if !ok {
			return "", errs.NewErrUnknownField(fd)
		}
		return fdMeta.ColName, nil

	default:
		return "", errs.NewErrUnsupportedExpressionType(tab)
	}
}

func (b *builder) quote(name string) {
	b.sb.WriteByte(b.quoter)
	b.sb.WriteString(name)
	b.sb.WriteByte(b.quoter)
}

func (b *builder) raw(r RawExpr) {
	b.sb.WriteString(r.raw)
	if len(r.args) != 0 {
		b.addArgs(r.args...)
	}
}

func (b *builder) addArgs(args ...any) {
	if b.args == nil {
		// 很少有查询能够超过八个参数
		// INSERT 除外
		b.args = make([]any, 0, 8)
	}
	b.args = append(b.args, args...)
}

func (b *builder) buildPredicates(ps []Predicate) error {
	p := ps[0]
	for i := 1; i < len(ps); i++ {
		p = p.And(ps[i])
	}
	return b.buildExpression(p)
}

func (b *builder) buildExpression(e Expression) error {
	if e == nil {
		return nil
	}
	switch exp := e.(type) {
	case Column:
		return b.buildColumn(exp.table, exp.name)
	case Aggregate:
		return b.buildAggregate(exp, false)
	case value:
		b.sb.WriteByte('?')
		b.addArgs(exp.val)
	case RawExpr:
		b.raw(exp)
	case MathExpr:
		return b.buildBinaryExpr(binaryExpr(exp))
	case Predicate:
		return b.buildBinaryExpr(binaryExpr(exp))
	case binaryExpr:
		return b.buildBinaryExpr(exp)
	case Subquery:
		return b.buildSubQuery(exp)
	case SubqueryExpr:
		b.sb.WriteString(exp.pred)
		b.buildSubQuery(exp.s)
	default:
		return errs.NewErrUnsupportedExpressionType(exp)
	}
	return nil
}

func (b *builder) buildSubQuery(e Subquery) error {

	if *e.subCacheSQL == "" {
		q, err := e.qb.Build()
		if err != nil {
			return err
		}
		*e.subCacheSQL = q.SQL[:len(q.SQL)-1]
		e.subCacheArgs = q.Args
	}

	b.sb.WriteByte('(')

	b.sb.WriteString(*e.subCacheSQL)

	if len(e.subCacheArgs) > 0 {
		b.args = append(b.args, e.subCacheArgs...)
	}

	b.sb.WriteByte(')')
	return nil
}

func (b *builder) buildBinaryExpr(e binaryExpr) error {
	err := b.buildSubExpr(e.left)
	if err != nil {
		return err
	}
	if e.op != "" {
		b.sb.WriteByte(' ')
		b.sb.WriteString(e.op.String())
	}
	if e.right != nil {
		b.sb.WriteByte(' ')
		return b.buildSubExpr(e.right)
	}
	return nil
}

func (b *builder) buildSubExpr(subExpr Expression) error {
	switch sub := subExpr.(type) {
	case MathExpr:
		_ = b.sb.WriteByte('(')
		if err := b.buildBinaryExpr(binaryExpr(sub)); err != nil {
			return err
		}
		_ = b.sb.WriteByte(')')
	case binaryExpr:
		_ = b.sb.WriteByte('(')
		if err := b.buildBinaryExpr(sub); err != nil {
			return err
		}
		_ = b.sb.WriteByte(')')
	case Predicate:
		_ = b.sb.WriteByte('(')
		if err := b.buildBinaryExpr(binaryExpr(sub)); err != nil {
			return err
		}
		_ = b.sb.WriteByte(')')
	default:
		if err := b.buildExpression(sub); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) buildAggregate(a Aggregate, useAlias bool) error {
	b.sb.WriteString(a.fn)
	b.sb.WriteByte('(')
	err := b.buildColumn(a.table, a.arg)
	if err != nil {
		return err
	}
	b.sb.WriteByte(')')
	if useAlias {
		b.buildAs(a.alias)
	}
	return nil
}

func (b *builder) buildAs(alias string) {
	if alias != "" {
		b.sb.WriteString(" AS ")
		b.quote(alias)
	}
}
