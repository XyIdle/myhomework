package homework_delete

import (
	"fmt"
	"reflect"
	"strings"
)

type Deleter[T any] struct {
	sb    strings.Builder
	args  []any
	table string
	where []Predicate
}

func (d *Deleter[T]) Build() (*Query, error) {
	d.sb.WriteString("DELETE FROM ")
	if d.table == "" {
		var t T
		d.sb.WriteByte('`')
		d.sb.WriteString(reflect.TypeOf(t).Name())
		d.sb.WriteByte('`')
	} else {
		d.sb.WriteString(d.table)
	}

	// 构造 WHERE
	if len(d.where) > 0 {
		// 类似这种可有可无的部分，都要在前面加一个空格
		d.sb.WriteString(" WHERE ")
		p := d.where[0]
		for i := 1; i < len(d.where); i++ {
			p = p.And(d.where[i])
		}
		if err := d.buildExpression(p); err != nil {
			return nil, err
		}
	}
	d.sb.WriteString(";")
	return &Query{
		SQL:  d.sb.String(),
		Args: d.args,
	}, nil
}

// From accepts model definition
func (d *Deleter[T]) From(table string) *Deleter[T] {
	d.table = table
	return d
}

// Where accepts predicates
func (d *Deleter[T]) Where(predicates ...Predicate) *Deleter[T] {
	d.where = predicates
	return d
}

func (d *Deleter[T]) buildExpression(e Expression) error {
	if e == nil {
		return nil
	}
	switch exp := e.(type) {
	case Column:
		d.sb.WriteByte('`')
		d.sb.WriteString(exp.name)
		d.sb.WriteByte('`')
	case value:
		d.sb.WriteByte('?')
		d.args = append(d.args, exp.val)
	case Predicate:
		_, lp := exp.left.(Predicate)
		if lp {
			d.sb.WriteByte('(')
		}
		if err := d.buildExpression(exp.left); err != nil {
			return err
		}
		if lp {
			d.sb.WriteByte(')')
		}

		d.sb.WriteByte(' ')
		d.sb.WriteString(exp.op.String())
		d.sb.WriteByte(' ')

		_, rp := exp.right.(Predicate)
		if rp {
			d.sb.WriteByte('(')
		}
		if err := d.buildExpression(exp.right); err != nil {
			return err
		}
		if rp {
			d.sb.WriteByte(')')
		}
	default:
		return fmt.Errorf("orm: 不支持的表达式 %v", exp)
	}
	return nil
}
