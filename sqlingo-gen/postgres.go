package main

import "database/sql"

type postgresSchemaFetcher struct {
	db *sql.DB
}

func (p postgresSchemaFetcher) GetDatabaseName() (dbName string, err error) {
	row := p.db.QueryRow("SELECT current_database()")
	err = row.Scan(&dbName)
	return
}

func (p postgresSchemaFetcher) GetTableNames() (tableNames []string, err error) {
	rows, err := p.db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public'")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return
		}
		tableNames = append(tableNames, name)
	}
	return
}

func (p postgresSchemaFetcher) GetFieldDescriptors(tableName string) (result []FieldDescriptor, err error) {
	rows, err := p.db.Query("SELECT column_name, is_nullable, data_type FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1", tableName)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var fieldDescriptor FieldDescriptor
		var isNullable string
		if err = rows.Scan(&fieldDescriptor.Name, &isNullable, &fieldDescriptor.Type); err != nil {
			return
		}
		fieldDescriptor.AllowNull = isNullable == "YES"
		result = append(result, fieldDescriptor)
	}
	return
}

func (p postgresSchemaFetcher) QuoteIdentifier(identifier string) string {
	return "\"" + identifier + "\""
}

func NewPostgresSchemaFetcher(db *sql.DB) SchemaFetcher {
	return postgresSchemaFetcher{db: db}
}
