package sqlingo

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type priority uint8

// Expression is the interface of an SQL expression.
type Expression interface {
	// get the SQL string
	GetSQL(scope scope) (string, error)
	getOperatorPriority() priority

	// <> operator
	NotEquals(other interface{}) BooleanExpression
	// == operator
	Equals(other interface{}) BooleanExpression
	// < operator
	LessThan(other interface{}) BooleanExpression
	// <= operator
	LessThanOrEquals(other interface{}) BooleanExpression
	// > operator
	GreaterThan(other interface{}) BooleanExpression
	// >= operator
	GreaterThanOrEquals(other interface{}) BooleanExpression

	IsNull() BooleanExpression
	IsNotNull() BooleanExpression
	IsTrue() BooleanExpression
	IsNotTrue() BooleanExpression
	IsFalse() BooleanExpression
	IsNotFalse() BooleanExpression
	In(values ...interface{}) BooleanExpression
	NotIn(values ...interface{}) BooleanExpression
	Between(min interface{}, max interface{}) BooleanExpression
	NotBetween(min interface{}, max interface{}) BooleanExpression
	Desc() OrderBy

	As(alias string) Alias

	If(trueValue interface{}, falseValue interface{}) UnknownExpression
	IfNull(altValue interface{}) UnknownExpression
}

// Alias is the interface of an table/column alias.
type Alias interface {
	GetSQL(scope scope) (string, error)
}

// BooleanExpression is the interface of an SQL expression with boolean value.
type BooleanExpression interface {
	Expression
	And(other interface{}) BooleanExpression
	Or(other interface{}) BooleanExpression
	Xor(other interface{}) BooleanExpression
	Not() BooleanExpression
}

// NumberExpression is the interface of an SQL expression with number value.
type NumberExpression interface {
	Expression
	Add(other interface{}) NumberExpression
	Sub(other interface{}) NumberExpression
	Mul(other interface{}) NumberExpression
	Div(other interface{}) NumberExpression
	IntDiv(other interface{}) NumberExpression
	Mod(other interface{}) NumberExpression

	Sum() NumberExpression
	Avg() NumberExpression
	Min() UnknownExpression
	Max() UnknownExpression
}

// StringExpression is the interface of an SQL expression with string value.
type StringExpression interface {
	Expression
	Min() UnknownExpression
	Max() UnknownExpression
	Like(other interface{}) BooleanExpression
	Contains(substring string) BooleanExpression
	Concat(other interface{}) StringExpression
	IfEmpty(altValue interface{}) StringExpression
	IsEmpty() BooleanExpression
	Lower() StringExpression
	Upper() StringExpression
	Left(count interface{}) StringExpression
	Right(count interface{}) StringExpression
	Trim() StringExpression
}

type ArrayExpression interface {
	Expression
}

type DateExpression interface {
	Expression
	Min() UnknownExpression
	Max() UnknownExpression
}

// UnknownExpression is the interface of an SQL expression with unknown value.
type UnknownExpression interface {
	Expression
	And(other interface{}) BooleanExpression
	Or(other interface{}) BooleanExpression
	Xor(other interface{}) BooleanExpression
	Not() BooleanExpression
	Add(other interface{}) NumberExpression
	Sub(other interface{}) NumberExpression
	Mul(other interface{}) NumberExpression
	Div(other interface{}) NumberExpression
	IntDiv(other interface{}) NumberExpression
	Mod(other interface{}) NumberExpression

	Sum() NumberExpression
	Avg() NumberExpression
	Min() UnknownExpression
	Max() UnknownExpression

	Like(other interface{}) BooleanExpression
	Contains(substring string) BooleanExpression
	Concat(other interface{}) StringExpression
	IfEmpty(altValue interface{}) StringExpression
	IsEmpty() BooleanExpression
	Lower() StringExpression
	Upper() StringExpression
	Left(count interface{}) StringExpression
	Right(count interface{}) StringExpression
	Trim() StringExpression
}

type expression struct {
	sql      string
	builder  func(scope scope) (string, error)
	priority priority
	isTrue   bool
	isFalse  bool
	isBool   bool
}

func (e expression) GetTable() Table {
	return nil
}

type scope struct {
	Database *database
	Tables   []Table
	lastJoin *join
}

func staticExpression(sql string, priority priority, isBool bool) expression {
	return expression{
		sql:      sql,
		priority: priority,
		isBool:   isBool,
	}
}

func True() BooleanExpression {
	return expression{
		sql:    "1",
		isTrue: true,
		isBool: true,
	}
}

func False() BooleanExpression {
	return expression{
		sql:     "0",
		isFalse: true,
		isBool:  true,
	}
}

// Raw create a raw SQL statement
func Raw(sql string) UnknownExpression {
	return expression{
		sql:      sql,
		priority: 99,
	}
}

// And creates an expression with AND operator.
func And(expressions ...BooleanExpression) (result BooleanExpression) {
	if len(expressions) == 0 {
		result = True()
		return
	}
	for _, condition := range expressions {
		if result == nil {
			result = condition
		} else {
			result = result.And(condition)
		}
	}
	return
}

// Or creates an expression with OR operator.
func Or(expressions ...BooleanExpression) (result BooleanExpression) {
	if len(expressions) == 0 {
		result = False()
		return
	}
	for _, condition := range expressions {
		if result == nil {
			result = condition
		} else {
			result = result.Or(condition)
		}
	}
	return
}

func (e expression) As(name string) Alias {
	return expression{builder: func(scope scope) (string, error) {
		expressionSql, err := e.GetSQL(scope)
		if err != nil {
			return "", err
		}
		return expressionSql + " AS " + name, nil
	}}
}

func (e expression) If(trueValue interface{}, falseValue interface{}) UnknownExpression {
	return If(e, trueValue, falseValue)
}

func (e expression) IfNull(altValue interface{}) UnknownExpression {
	return Function("IFNULL", e, altValue)
}

func (e expression) IfEmpty(altValue interface{}) StringExpression {
	return If(e.NotEquals(""), e, altValue)
}

func (e expression) IsEmpty() BooleanExpression {
	return e.Equals("")
}

func (e expression) Lower() StringExpression {
	return function("LOWER", e)
}

func (e expression) Upper() StringExpression {
	return function("UPPER", e)
}

func (e expression) Left(count interface{}) StringExpression {
	return function("LEFT", e, count)
}

func (e expression) Right(count interface{}) StringExpression {
	return function("RIGHT", e, count)
}

func (e expression) Trim() StringExpression {
	return function("TRIM", e)
}

func (e expression) CharLength() NumberExpression {
	return function("CHAR_LENGTH", e)
}

func (e expression) HasPrefix(prefix interface{}) BooleanExpression {
	return e.Left(function("CHAR_LENGTH", prefix)).Equals(prefix)
}

func (e expression) HasSuffix(suffix interface{}) BooleanExpression {
	return e.Right(function("CHAR_LENGTH", suffix)).Equals(suffix)
}

func (e expression) GetSQL(scope scope) (string, error) {
	if e.sql != "" {
		return e.sql, nil
	}
	return e.builder(scope)
}

var needsEscape = [256]int{
	0:    1,
	'\n': 1,
	'\r': 1,
	'\\': 1,
	'\'': 1,
	'"':  1,
	0x1a: 1,
}

func quoteIdentifier(identifier string) (result dialectArray) {
	for dialect := dialect(0); dialect < dialectCount; dialect++ {
		switch dialect {
		case dialectMySQL:
			result[dialect] = "`" + identifier + "`"
		case dialectMSSQL:
			result[dialect] = "[" + identifier + "]"
		default:
			result[dialect] = "\"" + identifier + "\""
		}
	}
	return
}

func quoteString(s string) string {
	if s == "" {
		return "''"
	}

	buf := make([]byte, len(s)*2+2)
	buf[0] = '\''
	n := 1
	for i := 0; i < len(s); i++ {
		b := s[i]
		buf[n] = '\\'
		n += needsEscape[b]
		buf[n] = b
		n++
	}
	buf[n] = '\''
	n++
	buf = buf[:n]
	return *(*string)(unsafe.Pointer(&buf))
}

func getSQL(scope scope, value interface{}) (sql string, priority priority, err error) {
	const mysqlTimeFormat = "2006-01-02 15:04:05.000000"
	if value == nil {
		sql = "NULL"
		return
	}
	switch value.(type) {
	case int:
		sql = strconv.Itoa(value.(int))
	case string:
		sql = quoteString(value.(string))
	case Expression:
		sql, err = value.(Expression).GetSQL(scope)
		priority = value.(Expression).getOperatorPriority()
	case Assignment:
		sql, err = value.(Assignment).GetSQL(scope)
	case toSelectFinal:
		sql, err = value.(toSelectFinal).GetSQL()
		if err != nil {
			return
		}
		sql = "(" + sql + ")"
	case toUpdateFinal:
		sql, err = value.(toUpdateFinal).GetSQL()
	case Table:
		sql = value.(Table).GetSQL(scope)
	case CaseExpression:
		sql, err = value.(CaseExpression).End().GetSQL(scope)
	case time.Time:
		tm := value.(time.Time)
		if tm.IsZero() {
			sql = "NULL"
		} else {
			tmStr := tm.Format(mysqlTimeFormat)
			sql = quoteString(tmStr)
		}
	case *time.Time:
		tm := value.(*time.Time)
		if tm == nil || tm.IsZero() {
			sql = "NULL"
		} else {
			tmStr := tm.Format(mysqlTimeFormat)
			sql = quoteString(tmStr)
		}
	default:
		v := reflect.ValueOf(value)
		sql, priority, err = getSQLFromReflectValue(scope, v)
	}
	return
}

func getSQLFromReflectValue(scope scope, v reflect.Value) (sql string, priority priority, err error) {
	if v.Kind() == reflect.Ptr {
		// dereference pointers
		for {
			if v.IsNil() {
				sql = "NULL"
				return
			}
			v = v.Elem()
			if v.Kind() != reflect.Ptr {
				break
			}
		}
		sql, priority, err = getSQL(scope, v.Interface())
		return
	}

	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			sql = "1"
		} else {
			sql = "0"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		sql = strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		sql = strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		sql = strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case reflect.String:
		sql = quoteString(v.String())
	case reflect.Array, reflect.Slice:
		length := v.Len()
		values := make([]interface{}, length)
		for i := 0; i < length; i++ {
			values[i] = v.Index(i).Interface()
		}
		sql, err = commaValues(scope, values)
		if err == nil {
			sql = "(" + sql + ")"
		}
	default:
		if vs, ok := v.Interface().(interface{ String() string }); ok {
			sql = quoteString(vs.String())
		} else {
			err = fmt.Errorf("invalid type %s", v.Kind().String())
		}
	}
	return
}

/*
1 INTERVAL
2 BINARY, COLLATE
3 !
4 - (unary minus), ~ (unary bit inversion)
5 ^
6 *, /, DIV, %, MOD
7 -, +
8 <<, >>
9 &
10 |
11 = (comparison), <=>, >=, >, <=, <, <>, !=, IS, LIKE, REGEXP, IN
12 BETWEEN, CASE, WHEN, THEN, ELSE
13 NOT
14 AND, &&
15 XOR
16 OR, ||
17 = (assignment), :=
*/
func (e expression) NotEquals(other interface{}) BooleanExpression {
	return e.binaryOperation("<>", other, 11, true)
}

func (e expression) Equals(other interface{}) BooleanExpression {
	return e.binaryOperation("=", other, 11, true)
}

func (e expression) LessThan(other interface{}) BooleanExpression {
	return e.binaryOperation("<", other, 11, true)
}

func (e expression) LessThanOrEquals(other interface{}) BooleanExpression {
	return e.binaryOperation("<=", other, 11, true)
}

func (e expression) GreaterThan(other interface{}) BooleanExpression {
	return e.binaryOperation(">", other, 11, true)
}

func (e expression) GreaterThanOrEquals(other interface{}) BooleanExpression {
	return e.binaryOperation(">=", other, 11, true)
}

func toBooleanExpression(value interface{}) BooleanExpression {
	e, ok := value.(expression)
	switch {
	case !ok:
		return nil
	case e.isTrue:
		return True()
	case e.isFalse:
		return False()
	case e.isBool:
		return e
	default:
		return nil
	}
}

func (e expression) And(other interface{}) BooleanExpression {
	switch {
	case e.isFalse:
		return e
	case e.isTrue:
		if exp := toBooleanExpression(other); exp != nil {
			return exp
		}
	}
	return e.binaryOperation("AND", other, 14, true)
}

func (e expression) Or(other interface{}) BooleanExpression {
	switch {
	case e.isTrue:
		return e
	case e.isFalse:
		if exp := toBooleanExpression(other); exp != nil {
			return exp
		}
	}
	return e.binaryOperation("OR", other, 16, true)
}

func (e expression) Xor(other interface{}) BooleanExpression {
	return e.binaryOperation("XOR", other, 15, true)
}

func (e expression) Add(other interface{}) NumberExpression {
	return e.binaryOperation("+", other, 7, false)
}

func (e expression) Sub(other interface{}) NumberExpression {
	return e.binaryOperation("-", other, 7, false)
}

func (e expression) Mul(other interface{}) NumberExpression {
	return e.binaryOperation("*", other, 6, false)
}

func (e expression) Div(other interface{}) NumberExpression {
	return e.binaryOperation("/", other, 6, false)
}

func (e expression) IntDiv(other interface{}) NumberExpression {
	return e.binaryOperation("DIV", other, 6, false)
}

func (e expression) Mod(other interface{}) NumberExpression {
	return e.binaryOperation("%", other, 6, false)
}

func (e expression) Sum() NumberExpression {
	return function("SUM", e)
}

func (e expression) Avg() NumberExpression {
	return function("AVG", e)
}

func (e expression) Min() UnknownExpression {
	return function("MIN", e)
}

func (e expression) Max() UnknownExpression {
	return function("MAX", e)
}

func (e expression) Like(other interface{}) BooleanExpression {
	return e.binaryOperation("LIKE", other, 11, true)
}

func (e expression) Concat(other interface{}) StringExpression {
	return Concat(e, other)
}

func (e expression) Contains(substring string) BooleanExpression {
	return function("LOCATE", substring, e).GreaterThan(0)
}

func (e expression) binaryOperation(operator string, value interface{}, priority priority, isBool bool) expression {
	return expression{builder: func(scope scope) (string, error) {
		leftSql, err := e.GetSQL(scope)
		if err != nil {
			return "", err
		}
		leftPriority := e.priority
		rightSql, rightPriority, err := getSQL(scope, value)
		if err != nil {
			return "", err
		}
		shouldParenthesizeLeft := leftPriority > priority
		shouldParenthesizeRight := rightPriority >= priority
		var sb strings.Builder
		sb.Grow(len(leftSql) + len(operator) + len(rightSql) + 6)
		if shouldParenthesizeLeft {
			sb.WriteByte('(')
		}
		sb.WriteString(leftSql)
		if shouldParenthesizeLeft {
			sb.WriteByte(')')
		}
		sb.WriteByte(' ')
		sb.WriteString(operator)
		sb.WriteByte(' ')
		if shouldParenthesizeRight {
			sb.WriteByte('(')
		}
		sb.WriteString(rightSql)
		if shouldParenthesizeRight {
			sb.WriteByte(')')
		}
		return sb.String(), nil
	}, priority: priority, isBool: isBool}
}

func (e expression) prefixSuffixExpression(prefix string, suffix string, priority priority, isBool bool) expression {
	if e.sql != "" {
		return expression{
			sql:      prefix + e.sql + suffix,
			priority: priority,
			isBool:   isBool,
		}
	}
	return expression{
		builder: func(scope scope) (string, error) {
			exprSql, err := e.GetSQL(scope)
			if err != nil {
				return "", err
			}
			var sb strings.Builder
			sb.Grow(len(prefix) + len(exprSql) + len(suffix) + 2)
			sb.WriteString(prefix)
			shouldParenthesize := e.priority > priority
			if shouldParenthesize {
				sb.WriteByte('(')
			}
			sb.WriteString(exprSql)
			if shouldParenthesize {
				sb.WriteByte(')')
			}
			sb.WriteString(suffix)
			return sb.String(), nil
		},
		priority: priority,
		isBool:   isBool,
	}
}

func (e expression) IsNull() BooleanExpression {
	return e.prefixSuffixExpression("", " IS NULL", 11, true)
}

func (e expression) Not() BooleanExpression {
	switch {
	case e.isTrue:
		return False()
	case e.isFalse:
		return True()
	default:
		return e.prefixSuffixExpression("NOT ", "", 13, true)
	}
}

func (e expression) IsNotNull() BooleanExpression {
	return e.prefixSuffixExpression("", " IS NOT NULL", 11, true)
}

func (e expression) IsTrue() BooleanExpression {
	return e.prefixSuffixExpression("", " IS TRUE", 11, true)
}

func (e expression) IsNotTrue() BooleanExpression {
	return e.prefixSuffixExpression("", " IS NOT TRUE", 11, true)
}

func (e expression) IsFalse() BooleanExpression {
	return e.prefixSuffixExpression("", " IS FALSE", 11, true)
}

func (e expression) IsNotFalse() BooleanExpression {
	return e.prefixSuffixExpression("", " IS NOT FALSE", 11, true)
}

func expandSliceValue(value reflect.Value) (result []interface{}) {
	result = make([]interface{}, 0, 16)
	kind := value.Kind()
	switch kind {
	case reflect.Array, reflect.Slice:
		length := value.Len()
		for i := 0; i < length; i++ {
			result = append(result, expandSliceValue(value.Index(i))...)
		}
	case reflect.Interface, reflect.Ptr:
		result = append(result, expandSliceValue(value.Elem())...)
	default:
		result = append(result, value.Interface())
	}
	return
}

func expandSliceValues(values []interface{}) (result []interface{}) {
	result = make([]interface{}, 0, 16)
	for _, v := range values {
		value := reflect.ValueOf(v)
		result = append(result, expandSliceValue(value)...)
	}
	return
}

func (e expression) In(values ...interface{}) BooleanExpression {
	values = expandSliceValues(values)
	if len(values) == 0 {
		return False()
	}
	joiner := func(exprSql, valuesSql string) string { return exprSql + " IN (" + valuesSql + ")" }
	builder := e.getBuilder(e.Equals, joiner, values...)
	return expression{builder: builder, priority: 11}
}

func (e expression) NotIn(values ...interface{}) BooleanExpression {
	values = expandSliceValues(values)
	if len(values) == 0 {
		return True()
	}
	joiner := func(exprSql, valuesSql string) string { return exprSql + " NOT IN (" + valuesSql + ")" }
	builder := e.getBuilder(e.NotEquals, joiner, values...)
	return expression{builder: builder, priority: 11}
}

type joinerFunc = func(exprSql, valuesSql string) string
type booleanFunc = func(other interface{}) BooleanExpression
type builderFunc = func(scope scope) (string, error)

func (e expression) getBuilder(single booleanFunc, joiner joinerFunc, values ...interface{}) builderFunc {
	return func(scope scope) (string, error) {
		var valuesSql string
		var err error

		if len(values) == 1 {
			value := values[0]
			if selectStatus, ok := value.(toSelectFinal); ok {
				// IN subquery
				valuesSql, err = selectStatus.GetSQL()
				if err != nil {
					return "", err
				}
			} else {
				// IN a single value
				return single(value).GetSQL(scope)
			}
		} else {
			// IN a list
			valuesSql, err = commaValues(scope, values)
			if err != nil {
				return "", err
			}
		}

		exprSql, err := e.GetSQL(scope)
		if err != nil {
			return "", err
		}
		return joiner(exprSql, valuesSql), nil
	}
}

func (e expression) Between(min interface{}, max interface{}) BooleanExpression {
	return e.buildBetween(" BETWEEN ", min, max)
}

func (e expression) NotBetween(min interface{}, max interface{}) BooleanExpression {
	return e.buildBetween(" NOT BETWEEN ", min, max)
}

func (e expression) buildBetween(operator string, min interface{}, max interface{}) BooleanExpression {
	return expression{builder: func(scope scope) (string, error) {
		exprSql, err := e.GetSQL(scope)
		if err != nil {
			return "", err
		}
		minSql, _, err := getSQL(scope, min)
		if err != nil {
			return "", err
		}
		maxSql, _, err := getSQL(scope, max)
		if err != nil {
			return "", err
		}
		return exprSql + operator + minSql + " AND " + maxSql, nil
	}, priority: 12}
}

func (e expression) getOperatorPriority() priority {
	return e.priority
}

func (e expression) Desc() OrderBy {
	return orderBy{by: e, desc: true}
}
