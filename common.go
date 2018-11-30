package sqlingo

import (
	"fmt"
	"runtime"
	"strings"
)

type Model interface {
	GetTable() Table
	GetValues() []interface{}
}

type Assignment interface {
	GetSQL(scope scope) (string, error)
}

type assignment struct {
	Assignment
	field Field
	value interface{}
}

func (a assignment) GetSQL(scope scope) (string, error) {
	value, _, err := getSQLFromWhatever(scope, a.value)
	if err != nil {
		return "", err
	}
	fieldSql, err := a.field.GetSQL(scope)
	if err != nil {
		return "", err
	}
	return fieldSql + " = " + value, nil
}

func Raw(sql string) UnknownExpression {
	return staticExpression(sql, 99)
}

func And(expressions ...BooleanExpression) (result BooleanExpression) {
	if len(expressions) == 0 {
		result = staticExpression("1", 0)
		return
	}
	for i, condition := range expressions {
		if i == 0 {
			result = condition
		} else {
			result = result.And(condition)
		}
	}
	return
}

func Or(expressions ...BooleanExpression) (result BooleanExpression) {
	if len(expressions) == 0 {
		result = staticExpression("0", 0)
		return
	}
	for i, condition := range expressions {
		if i == 0 {
			result = condition
		} else {
			result = result.Or(condition)
		}
	}
	return
}

func function(name string, args ...interface{}) expression {
	return expression{builder: func(scope scope) (string, error) {
		valuesSql, err := commaValues(scope, args)
		if err != nil {
			return "", err
		}
		return name + "(" + valuesSql + ")", nil
	}}
}

func Function(name string, args ...interface{}) Expression {
	return function(name, args...)
}

func command(args ...interface{}) expression {
	return expression{builder: func(scope scope) (string, error) {
		sql := ""
		for i, item := range args {
			if i > 0 {
				sql += " "
			}
			itemSql, _, err := getSQLFromWhatever(scope, item)
			if err != nil {
				return "", err
			}
			sql += itemSql

		}
		return sql, nil
	}}
}

func If(predicate Expression, trueValue Expression, falseValue Expression) (result Expression) {
	return Function("IF", predicate, trueValue, falseValue)
}

func commaFields(scope scope, fields []Field) (string, error) {
	sql := ""
	for i, item := range fields {
		if i > 0 {
			sql += ", "
		}
		itemSql, err := item.GetSQL(scope)
		if err != nil {
			return "", err
		}
		sql += itemSql
	}
	return sql, nil
}

func commaExpressions(scope scope, expressions []Expression) (string, error) {
	sql := ""
	for i, item := range expressions {
		if i > 0 {
			sql += ", "
		}
		itemSql, err := item.GetSQL(scope)
		if err != nil {
			return "", err
		}
		sql += itemSql
	}
	return sql, nil
}

func commaValues(scope scope, values []interface{}) (string, error) {
	sql := ""
	for i, item := range values {
		if i > 0 {
			sql += ", "
		}
		itemSql, _, err := getSQLFromWhatever(scope, item)
		if err != nil {
			return "", err
		}
		sql += itemSql
	}
	return sql, nil
}

func commaAssignments(scope scope, assignments []assignment) (string, error) {
	sql := ""
	for i, item := range assignments {
		if i > 0 {
			sql += ", "
		}
		itemSql, err := item.GetSQL(scope)
		if err != nil {
			return "", err
		}
		sql += itemSql
	}
	return sql, nil
}

func commaOrderBys(scope scope, orderBys []OrderBy) (string, error) {
	sql := ""
	for i, item := range orderBys {
		if i > 0 {
			sql += ", "
		}
		itemSql, err := item.GetSQL(scope)
		if err != nil {
			return "", err
		}
		sql += itemSql
	}
	return sql, nil
}

func getSQLForName(name string) string {
	// TODO: check reserved words
	return "`" + name + "`"
}

func getCallerInfo(db Database) string {
	txInfo := ""
	switch db.(type) {
	case *database:
		if db.(*database).tx != nil {
			txInfo = " (tx)"
		}
	}
	for i := 0; true; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if strings.Contains(file, "/sqlingo@v") {
			continue
		}
		segs := strings.Split(file, "/")
		name := segs[len(segs)-1]
		return fmt.Sprintf("/* %s:%d%s */ ", name, line, txInfo)
	}
	return ""
}
