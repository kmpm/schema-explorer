package reader

import (
	"bitbucket.org/timabell/sql-data-viewer/options"
	"bitbucket.org/timabell/sql-data-viewer/params"
	"bitbucket.org/timabell/sql-data-viewer/schema"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
)

type DbReader interface {
	// does select or something to make sure we have a working db connection
	CheckConnection() (err error)

	// parse the whole schema info into memory
	ReadSchema() (database *schema.Database, err error)

	// populate the table row counts
	UpdateRowCounts(database *schema.Database) (err error)

	// get some data, obeying sorting, filtering etc in the table params
	GetSqlRows(table *schema.Table, params *params.TableParams, peekFinder *PeekLookup) (rows *sql.Rows, err error)

	// get a count for the supplied filters, for use with paging and overview info
	GetRowCount(table *schema.Table, params *params.TableParams) (rowCount int, err error)

	// get breakdown of most common values in each column
	GetAnalysis(table *schema.Table) (analysis []schema.ColumnAnalysis, err error)
}

type CreateReader func() DbReader

// Single row of data
type RowData []interface{}

var creators = make(map[string]CreateReader)

// This is how implementations for reading different RDBMS systems can register themselves.
// They should call this in their init() function
func RegisterReader(name string, opt interface{}, creator CreateReader) {
	creators[name] = creator
	group, err := options.ArgParser.AddGroup(name, fmt.Sprintf("Options for %s database", name), opt)
	if err != nil {
		panic(err)
	}
	group.Namespace = name
	group.EnvNamespace = name
}

func GetDbReader() DbReader {
	if options.Options == nil || (*options.Options).Driver == nil {
		panic("driver option missing")
	}
	createReader := creators[*options.Options.Driver]
	if createReader == nil {
		log.Printf("Unknown reader '%s'", *options.Options.Driver)
		os.Exit(1)
	}
	return createReader()
}

func GetRows(reader DbReader, table *schema.Table, params *params.TableParams) (rowsData []RowData, peekFinder *PeekLookup, err error) {
	// load up all the fks that we have peek info for
	peekFinder = &PeekLookup{}
	inboundPeekCount := 0
	for _, fk := range table.Fks {
		if len(fk.DestinationTable.PeekColumns) == 0 {
			continue
		}
		peekFinder.Fks = append(peekFinder.Fks, fk)
		inboundPeekCount += len(fk.DestinationTable.PeekColumns)
	}
	peekFinder.outboundPeekStartIndex = len(table.Columns)
	peekFinder.inboundPeekStartIndex = peekFinder.outboundPeekStartIndex + inboundPeekCount
	peekFinder.peekColumnCount = inboundPeekCount + len(table.InboundFks)
	peekFinder.Table = table

	rows, err := reader.GetSqlRows(table, params, peekFinder)
	if rows == nil {
		panic("GetSqlRows() returned nil")
	}
	defer rows.Close()
	if len(table.Columns) == 0 {
		panic("No columns found when reading table data table")
	}
	rowsData, err = getAllData(len(table.Columns)+peekFinder.peekColumnCount, rows)
	if err != nil {
		return nil, nil, err
	}
	return
}

func getAllData(colCount int, rows *sql.Rows) (rowsData []RowData, err error) {
	for rows.Next() {
		row, err := getRow(colCount, rows)
		if err != nil {
			return nil, err
		}
		rowsData = append(rowsData, row)
	}
	return
}

func getRow(colCount int, rows *sql.Rows) (rowsData RowData, err error) {
	// http://stackoverflow.com/a/23507765/10245 - getting ad-hoc column data
	singleRow := make([]interface{}, colCount)
	rowDataPointers := make([]interface{}, colCount)
	for i := 0; i < colCount; i++ {
		rowDataPointers[i] = &singleRow[i]
	}
	err = rows.Scan(rowDataPointers...)
	if err != nil {
		log.Println("error reading row data", err)
		return nil, err
	}
	return singleRow, err
}

func DbValueToString(colData interface{}, dataType string) *string {
	var stringValue string
	uuidLen := 16
	switch {
	case colData == nil:
		return nil
	case dataType == "money": // mssql money
		fallthrough
	case dataType == "decimal": // mssql decimal
		fallthrough
	case dataType == "numeric": // mssql numeric
		stringValue = fmt.Sprintf("%s", colData) // seems to come back as byte array for a string, surprising, could be a driver thing
	case dataType == "integer":
		fallthrough
	case dataType == "int4":
		stringValue = fmt.Sprintf("%d", colData)
	case dataType == "float":
		stringValue = fmt.Sprintf("%f", colData)
	case dataType == "uniqueidentifier": // mssql guid
		bytes := colData.([]byte)
		if len(bytes) != uuidLen {
			panic(fmt.Sprintf("Unexpected byte-count for uniqueidentifier, expected %d, got %d. Value: %+v", uuidLen, len(bytes), colData))
		}
		stringValue = fmt.Sprintf("%x%x%x%x-%x%x-%x%x-%x%x-%x%x%x%x%x%x",
			bytes[3], bytes[2], bytes[1], bytes[0], bytes[5], bytes[4], bytes[7], bytes[6], bytes[8], bytes[9], bytes[10], bytes[11], bytes[12], bytes[13], bytes[14], bytes[15])
	case dataType == "text": // sqlite
		fallthrough
	case dataType == "jsonb": // pg
		fallthrough
	case dataType == "json": // pg
		fallthrough
	case dataType == "CLOB": // sqlite
		fallthrough
	case strings.Contains(strings.ToLower(dataType), "char"): // sqlite is uppercase, mssql is lowercase. See test data for things this should cover
		stringValue = fmt.Sprintf("%s", colData)
	case strings.Contains(dataType, "TEXT"): // mssql
		// https://stackoverflow.com/a/18615786/10245
		bytes := colData.([]uint8)
		stringValue = fmt.Sprintf("%s", bytes)
	case dataType == "varbinary": // mssql varbinary
		stringValue = "[binary]"
	default:
		log.Printf("unknown data type %s", dataType)
		//panic(fmt.Sprintf("unknown data type %s", dataType))
		stringValue = fmt.Sprintf("%v", colData)
	}
	return &stringValue
}
