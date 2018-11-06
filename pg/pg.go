package pg

import (
	"bitbucket.org/timabell/sql-data-viewer/params"
	"bitbucket.org/timabell/sql-data-viewer/reader"
	"bitbucket.org/timabell/sql-data-viewer/schema"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"os"
	"strconv"
	"strings"
)

type pgModel struct {
	connectionString string
}

type pgOpts struct {
	Host             *string `long:"host" description:"Postgres host" env:"host"`
	Port             *int    `long:"port" description:"Postgres port" env:"port"`
	Database         *string `long:"database" description:"Postgres database name" env:"database"`
	User             *string `long:"user" description:"Postgres username" env:"user"`
	Password         *string `long:"password" description:"Postgres password" env:"password"`
	ConnectionString *string `long:"connection-string" description:"Postgres connection string. Use this instead of host, port etc for advanced driver options. See https://godoc.org/github.com/lib/pq for connection-string options." env:"connection_string"`
}

func (opts pgOpts) validate() error {
	if opts.hasAnyDetails() && opts.ConnectionString != nil {
		return errors.New("Specify either a connection string or host etc, not both.")
	}
	return nil
}

func (opts pgOpts) hasAnyDetails() bool {
	return opts.Host != nil ||
		opts.Port != nil ||
		opts.Database != nil ||
		opts.User != nil ||
		opts.Password != nil
}

var opts = &pgOpts{}

func init() {
	// https://github.com/jessevdk/go-flags/blob/master/group_test.go#L33
	reader.RegisterReader("pg", opts, NewPg)
}

func NewPg() reader.DbReader {
	err := opts.validate()
	if err != nil {
		log.Printf("Pg args error: %s", err)
		reader.ArgParser.WriteHelp(os.Stdout)
		os.Exit(1)
	}
	var cs string
	if opts.ConnectionString == nil {
		optList := make(map[string]string)
		if opts.Host != nil {
			optList["host"] = *opts.Host
		}
		if opts.Port != nil {
			optList["port"] = strconv.Itoa(*opts.Port)
		}
		if opts.Database != nil {
			optList["dbname"] = *opts.Database
		}
		if opts.User != nil {
			optList["user"] = *opts.User
		}
		if opts.Password != nil {
			optList["password"] = *opts.Password
		}
		pairs := []string{}
		for key, value := range optList {
			pairs = append(pairs, fmt.Sprintf("%s='%s'", key, strings.Replace(value, "'", "\\'", -1)))
		}
		cs = strings.Join(pairs, " ")
	} else {
		cs = *opts.ConnectionString
	}
	log.Println("Connecting to pg db")
	return pgModel{
		connectionString: cs,
	}
}

func (model pgModel) ReadSchema() (database *schema.Database, err error) {
	dbc, err := getConnection(model.connectionString)
	if err != nil {
		return
	}
	defer dbc.Close()

	database = &schema.Database{
		Supports: schema.SupportedFeatures{
			Schema:       true,
			Descriptions: false,
			FkNames:      true,
		},
		DefaultSchemaName: "public",
	}

	// load table list
	database.Tables, err = model.getTables(dbc)
	if err != nil {
		return
	}

	// add table columns
	for _, table := range database.Tables {
		var cols []*schema.Column
		cols, err = model.getColumns(dbc, table)
		if err != nil {
			return
		}
		table.Columns = append(table.Columns, cols...)
	}

	// fks and other constraints
	err = readConstraints(dbc, database)
	if err != nil {
		return
	}

	// indexes
	err = readIndexes(dbc, database)
	if err != nil {
		return
	}

	//log.Print(database.DebugString())
	return
}

func (model pgModel) UpdateRowCounts(database *schema.Database) (err error) {
	for _, table := range database.Tables {
		rowCount, err := model.getRowCount(table)
		if err != nil {
			log.Printf("Failed to get row count for %s, %s", table, err)
		}
		table.RowCount = &rowCount
	}
	return err
}

func (model pgModel) getRowCount(table *schema.Table) (rowCount int, err error) {
	// todo: parameterise where possible
	// todo: whitelist-sanitize unparameterizable parts
	sql := "select count(*) from \"" + table.Name + "\""

	dbc, err := getConnection(model.connectionString)
	if dbc == nil {
		log.Println(err)
		panic("getConnection() returned nil")
	}
	defer dbc.Close()
	rows, err := dbc.Query(sql)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	rows.Next()
	var count int
	rows.Scan(&count)
	return count, nil
}

func (model pgModel) getTables(dbc *sql.DB) (tables []*schema.Table, err error) {
	rows, err := dbc.Query("select schemaname, tablename from pg_catalog.pg_tables where schemaname not in ('pg_catalog','information_schema') order by schemaname, tablename")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name, schemaName string
		rows.Scan(&schemaName, &name)
		tables = append(tables, &schema.Table{Schema: schemaName, Name: name, Pk: &schema.Pk{}})
	}
	return tables, nil
}

func getConnection(connectionString string) (dbc *sql.DB, err error) {
	dbc, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Println("connection error", err)
	}
	return
}

func (model pgModel) CheckConnection() (err error) {
	dbc, err := getConnection(model.connectionString)
	if dbc == nil {
		log.Println(err)
		panic("getConnection() returned nil")
	}
	defer dbc.Close()
	tables, err := model.getTables(dbc)
	if err != nil {
		panic(err)
	}
	if len(tables) == 0 {
		panic("No tables found.")
	}
	log.Println("Connected.", len(tables), "tables found")
	return
}

func readConstraints(dbc *sql.DB, database *schema.Database) (err error) {
	// null-proof unnest: https://stackoverflow.com/a/49736694
	sql := fmt.Sprintf(`
		select
			con.oid, ns.nspname, con.conname, con.contype,
			tns.nspname, tbl.relname, col.attname column_name,
			fns.nspname foreign_namespace_name, ftbl.relname foreign_table_name, fcol.attname foreign_column_name
		from
			(
				select pgc.oid, pgc.connamespace, pgc.conrelid, pgc.confrelid, pgc.contype, pgc.conname,
				       unnest(case when pgc.conkey <> '{}' then pgc.conkey else '{null}' end) as conkey,
				       unnest(case when pgc.confkey <> '{}' then pgc.confkey else '{null}' end) as confkey
				from pg_constraint pgc
			) as con
			inner join pg_namespace ns on con.connamespace = ns.oid
			inner join pg_class tbl on tbl.oid = con.conrelid
			inner join pg_namespace tns on tbl.relnamespace = tns.oid
			inner join pg_attribute col on col.attrelid = tbl.oid and col.attnum = con.conkey
			left outer join pg_class ftbl on ftbl.oid = con.confrelid
			left outer join pg_namespace fns on ftbl.relnamespace = fns.oid
			left outer join pg_attribute fcol on fcol.attrelid = ftbl.oid and fcol.attnum = con.confkey;`)

	rows, err := dbc.Query(sql)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var oid, conType, namespace, name,
			sourceNamespace, sourceTableName, sourceColumnName,
			destinationNamespace, destinationTableName, destinationColumnName string
		rows.Scan(&oid, &namespace, &name, &conType,
			&sourceNamespace, &sourceTableName, &sourceColumnName,
			&destinationNamespace, &destinationTableName, &destinationColumnName)
		tableToFind := &schema.Table{Schema: sourceNamespace, Name: sourceTableName}
		sourceTable := database.FindTable(tableToFind)
		if sourceTable == nil {
			err = errors.New(fmt.Sprintf("Table %s not found, source of constraint %s", tableToFind.String(), name))
			return
		}
		_, sourceColumn := sourceTable.FindColumn(sourceColumnName)
		if sourceColumn == nil {
			err = errors.New(fmt.Sprintf("Column %s not found on table %s, source of constraint %s", sourceColumnName, tableToFind.String(), name))
			return
		}
		switch conType {
		case "f": // f = foreign key
			destinationTable := database.FindTable(&schema.Table{Schema: destinationNamespace, Name: destinationTableName})
			if destinationTable == nil {
				//log.Print(database.DebugString())
				panic(fmt.Sprintf("couldn't find table %s in database object while hooking up fks", destinationTableName))
			}
			_, destinationColumn := destinationTable.FindColumn(destinationColumnName)
			// see if we are adding columns to an existing fk

			var fk *schema.Fk
			for _, existingFk := range database.Fks {
				if existingFk.Name == name {
					existingFk.SourceColumns = append(existingFk.SourceColumns, sourceColumn)
					existingFk.DestinationColumns = append(existingFk.DestinationColumns, destinationColumn)
					fk = existingFk
					break
				}
			}
			if fk == nil { // then this is a never-before-seen fk
				fk = schema.NewFk(name, sourceTable, sourceColumn, destinationTable, destinationColumn)
				database.Fks = append(database.Fks, fk)
				sourceTable.Fks = append(sourceTable.Fks, fk)
				sourceColumn.Fks = append(sourceColumn.Fks, fk)
				destinationTable.InboundFks = append(destinationTable.InboundFks, fk)
				destinationColumn.InboundFks = append(destinationColumn.InboundFks, fk)
			}
			//log.Printf("fk: %+v - oid %+v", fk, oid)
		case "p": // primary key
			//log.Printf("pk: %s.%s", sourceTable, sourceColumn)
			sourceTable.Pk.Columns = append(sourceTable.Pk.Columns, sourceColumn)
			sourceColumn.IsInPrimaryKey = true
		case "c": // todo: check constraint
		case "u": // todo: unique constraint
		case "t": // todo: constraint
		case "x": // todo: exclusion constraint
		default:
			log.Printf("?? %s", conType)
		}
	}
	return
}

func readIndexes(dbc *sql.DB, database *schema.Database) (err error) {
	sql := `
		select
			oc.relname,
			tns.nspname, tbl.relname table_relname,
			col.attname colname,
			ix.indisunique,
			ix.indisclustered
		from (
			select *, unnest(indkey) colnum from pg_index
		) ix
		left outer join pg_class oc on oc.oid = ix.indexrelid
		left outer join pg_class tbl on tbl.oid = ix.indrelid
		left outer join pg_namespace tns on tbl.relnamespace = tns.oid
		left outer join pg_attribute col on col.attrelid = ix.indrelid and col.attnum = ix.colnum
		where tns.nspname not like 'pg_%'
			and not ix.indisprimary;
	`

	//log.Println(sql)
	rows, err := dbc.Query(sql)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name, tableNamespace, tableName, colName string
		var isUnique, isClustered bool
		rows.Scan(&name, &tableNamespace, &tableName, &colName, &isUnique, &isClustered)
		tableToFind := &schema.Table{Schema: tableNamespace, Name: tableName}
		table := database.FindTable(tableToFind)
		if table == nil {
			err = errors.New(fmt.Sprintf("Table %s not found, owner of index %s", tableToFind.String(), name))
			return
		}
		var index *schema.Index
		for _, existingIndex := range table.Indexes {
			if existingIndex.Name == name {
				index = existingIndex
				break
			}
		}
		if index == nil {
			index = &schema.Index{
				Name:        name,
				Columns:     []*schema.Column{},
				IsUnique:    isUnique,
				Table:       table,
				IsClustered: isClustered,
			}
			database.Indexes = append(database.Indexes, index)
			table.Indexes = append(table.Indexes, index)
		}
		if colName != "" { // more complex indexes don't link back to their columns. See pg_index.indkey https://www.postgresql.org/docs/current/static/catalog-pg-index.html
			_, col := table.FindColumn(colName)
			if col == nil {
				err = errors.New(fmt.Sprintf("Column %s in table %s not found, for index %s", colName, tableToFind.String(), name))
				return
			}
			index.Columns = append(index.Columns, col)
			col.Indexes = append(col.Indexes, index)
		}
		//log.Printf(index.String())
	}
	return
}

func (model pgModel) GetSqlRows(table *schema.Table, params *params.TableParams) (rows *sql.Rows, err error) {
	// todo: parameterise where possible
	// todo: whitelist-sanitize unparameterizable parts
	sql := "select * from \"" + table.Name + "\""

	var values []interface{}
	query := params.Filter
	if len(query) > 0 {
		sql = sql + " where "
		clauses := make([]string, 0, len(query))
		values = make([]interface{}, 0, len(query))
		var index = 1
		for _, v := range query {
			col := v.Field
			clauses = append(clauses, "\""+col.Name+"\" = $"+strconv.Itoa(index))
			index = index + 1
			values = append(values, v.Values[0]) // todo: maybe support multiple values
		}
		sql = sql + strings.Join(clauses, " and ")
	}

	if len(params.Sort) > 0 {
		var sortParts []string
		for _, sortCol := range params.Sort {
			sortString := "\"" + sortCol.Column.Name + "\""
			if sortCol.Descending {
				sortString = sortString + " desc"
			}
			sortParts = append(sortParts, sortString)
		}
		sql = sql + " order by " + strings.Join(sortParts, ", ")
	}

	if params.RowLimit > 0 || params.SkipRows > 0 {
		sql = sql + fmt.Sprintf(" limit %d offset %d", params.RowLimit, params.SkipRows)
	}

	dbc, err := getConnection(model.connectionString)
	if err != nil {
		log.Print("GetRows failed to get connection")
		return
	}
	defer dbc.Close()

	log.Println(sql)
	rows, err = dbc.Query(sql, values...)
	if err != nil {
		log.Print("GetRows failed to get query")
		log.Println(sql)
		log.Println(err)
	}
	return
}

func (model pgModel) getColumns(dbc *sql.DB, table *schema.Table) (cols []*schema.Column, err error) {
	// todo: parameterise
	sql := "select col.attname colname, col.attlen, typ.typname, col.attnotnull from pg_catalog.pg_attribute col inner join pg_catalog.pg_class tbl on col.attrelid = tbl.oid inner join pg_catalog.pg_namespace ns on ns.oid = tbl.relnamespace inner join pg_catalog.pg_type typ on typ.oid = col.atttypid where col.attnum > 0 and not col.attisdropped and ns.nspname = '" + table.Schema + "' and tbl.relname = '" + table.Name + "' order by col.attnum;"

	rows, err := dbc.Query(sql)
	if err != nil {
		log.Print(sql)
		return
	}
	defer rows.Close()
	cols = []*schema.Column{}
	colIndex := 0
	for rows.Next() {
		var len int
		var name, typeName string
		var notNull bool
		rows.Scan(&name, &len, &typeName, &notNull)
		thisCol := schema.Column{Position: colIndex, Name: name, Type: typeName, Nullable: !notNull}
		cols = append(cols, &thisCol)
		colIndex++
	}
	return
}
