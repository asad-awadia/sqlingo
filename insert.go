package sqlingo

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

type insertStatus struct {
	method                          string
	scope                           scope
	fields                          []Field
	values                          []interface{}
	models                          []interface{}
	onDuplicateKeyUpdateAssignments []assignment
}

type InsertWithTable interface {
	Fields(fields ...Field) InsertWithValues
	Values(values ...interface{}) InsertWithValues
	Models(models ...interface{}) InsertWithModels
}

type InsertWithValues interface {
	Values(values ...interface{}) InsertWithValues
	OnDuplicateKeyUpdate() InsertWithOnDuplicateKeyUpdateBegin
	GetSQL() (string, error)
	Execute() (result sql.Result, err error)
}

type InsertWithModels interface {
	Models(models ...interface{}) InsertWithModels
	OnDuplicateKeyUpdate() InsertWithOnDuplicateKeyUpdateBegin
	GetSQL() (string, error)
	Execute() (result sql.Result, err error)
}

type InsertWithOnDuplicateKeyUpdateBegin interface {
	Set(Field Field, value interface{}) InsertWithOnDuplicateKeyUpdate
}

type InsertWithOnDuplicateKeyUpdate interface {
	Set(Field Field, value interface{}) InsertWithOnDuplicateKeyUpdate
	GetSQL() (string, error)
	Execute() (result sql.Result, err error)
}

func (d *database) InsertInto(table Table) InsertWithTable {
	return insertStatus{method: "INSERT", scope: scope{Database: d, Tables: []Table{table}}}
}

func (d *database) ReplaceInto(table Table) InsertWithTable {
	return insertStatus{method: "REPLACE", scope: scope{Database: d, Tables: []Table{table}}}
}

func (s insertStatus) Fields(fields ...Field) InsertWithValues {
	s.fields = fields
	return s
}

func (s insertStatus) Values(values ...interface{}) InsertWithValues {
	s.values = append([]interface{}{}, s.values...)
	s.values = append(s.values, values)
	return s
}

func addModel(models *[]Model, model interface{}) error {
	if model, ok := model.(Model); ok {
		*models = append(*models, model)
		return nil
	}

	value := reflect.ValueOf(model)
	switch value.Kind() {
	case reflect.Ptr:
		value = reflect.Indirect(value)
		return addModel(models, value.Interface())
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			elem := value.Index(i)
			addr := elem.Addr()
			inter := addr.Interface()
			if err := addModel(models, inter); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown model type (kind = %d)", value.Kind())
	}
}

func (s insertStatus) Models(models ...interface{}) InsertWithModels {
	s.models = models
	return s
}

func (s insertStatus) OnDuplicateKeyUpdate() InsertWithOnDuplicateKeyUpdateBegin {
	return s
}

func (s insertStatus) Set(field Field, value interface{}) InsertWithOnDuplicateKeyUpdate {
	s.onDuplicateKeyUpdateAssignments = append([]assignment{}, s.onDuplicateKeyUpdateAssignments...)
	s.onDuplicateKeyUpdateAssignments = append(s.onDuplicateKeyUpdateAssignments, assignment{
		field: field,
		value: value,
	})
	return s
}

func (s insertStatus) GetSQL() (string, error) {
	var fields []Field
	var values []interface{}
	if len(s.models) > 0 {
		models := make([]Model, 0, len(s.models))
		for _, model := range s.models {
			if err := addModel(&models, model); err != nil {
				return "", err
			}
		}

		fields = models[0].GetTable().GetFields()
		for _, model := range models {
			if model.GetTable().GetName() != s.scope.Tables[0].GetName() {
				return "", errors.New("invalid table from model")
			}
			values = append(values, model.GetValues())
		}
	} else {
		fields = s.fields
		values = s.values
	}

	tableSql := s.scope.Tables[0].GetSQL(s.scope)
	fieldsSql, err := commaFields(s.scope, fields)
	if err != nil {
		return "", err
	}
	valuesSql, err := commaValues(s.scope, values)
	if err != nil {
		return "", err
	}

	sqlString := s.method + " INTO " + tableSql + " (" + fieldsSql + ") VALUES " + valuesSql
	if len(s.onDuplicateKeyUpdateAssignments) > 0 {
		assignmentsSql, err := commaAssignments(s.scope, s.onDuplicateKeyUpdateAssignments)
		if err != nil {
			return "", err
		}
		sqlString += " ON DUPLICATE KEY UPDATE " + assignmentsSql
	}

	return sqlString, nil
}

func (s insertStatus) Execute() (result sql.Result, err error) {
	sqlString, err := s.GetSQL()
	if err != nil {
		return nil, err
	}
	return s.scope.Database.Execute(sqlString)
}
