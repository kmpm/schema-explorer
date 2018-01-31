package sdv

/*
Tests multiple rdbms implementations by way of test.sh shell script that repeatedly runs the same
tests for each supported rdbms.
Relies on matching sql files having been run to set up each test database.

The tests are testing pulling data from a real database (integration testing) because
the layer between the code and the database is the most fragile.
The tests do not cover the UI layer beyond translation of data from the database into
strings for display.

In order to test different databases where they support an overlapping but not identical
set of data types the following strategy is used:

Every supported db system gets a table with a column for each data type that is supported by
that rdbms, named to match, then the test code tests the conversion to string for each
available data type. This allows data types that are common to be tested with a single test
but still support data types that are unique to a particular rdbms.

The expected number of cols is included in an extra column so we can double-check that we
aren't silently missing any of the supported data types.
*/

import (
	"flag"
	"testing"

	"bitbucket.org/timabell/sql-data-viewer/schema"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/simnalamburt/go-mssqldb"
	"strings"
)

var testDb string
var testDbDriver string

func init() {
	flag.StringVar(&testDbDriver, "driver", "", "Driver to use (mssql or sqlite)")
	flag.StringVar(&testDb, "db", "", "connection string for mssql / filename for sqlite")
	flag.Parse()
	if testDbDriver == "" {
		flag.Usage()
		panic("Driver argument required.")
	}
	if testDb == "" {
		flag.Usage()
		panic("Db argument required.")
	}
}

func Test_CheckConnection(t *testing.T) {
	reader := getDbReader(testDbDriver, testDb)
	err := reader.CheckConnection()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_ReadSchema(t *testing.T) {
	reader := getDbReader(testDbDriver, testDb)
	database, err := reader.ReadSchema()
	if err != nil {
		t.Fatal(err)
	}

	checkFkCount(database, t)
	checkTableFkCounts(database, t)
}

func checkFkCount(database schema.Database, t *testing.T) {
	expectedCount := 1
	fkCount := len(database.Fks)
	if fkCount != expectedCount {
		t.Fatalf("Expected %d fks across whole db, found %d", expectedCount, fkCount)
	}
}

func checkTableFkCounts(database schema.Database, t *testing.T) {
	checkTableFkCount("person", database, t)
	checkTableFkCount("pet", database, t)
}

func checkTableFkCount(tableName string, database schema.Database, t *testing.T) {
	expectedCount := 1
	table := findTable(tableName, database, t)
	fkCount := len(table.Fks)
	if fkCount != expectedCount {
		t.Fatalf("Expected %d fks in table %s, found %d", expectedCount, table, fkCount)
	}
}

type testCase struct {
	colName        string
	row            int
	expectedType   string
	expectedString string
}

var tests = []testCase{
	{colName: "field_INT", row: 0, expectedType: "int", expectedString: "20"},
	{colName: "field_INT", row: 1, expectedType: "int", expectedString: "-33"},
	{colName: "field_money", row: 0, expectedType: "money", expectedString: "1234.5670"},
	{colName: "field_numeric", row: 0, expectedType: "numeric", expectedString: "987.1234500"},
	{colName: "field_decimal", row: 0, expectedType: "decimal", expectedString: "666.1234500"},
	{colName: "field_uniqueidentifier", row: 0, expectedType: "uniqueidentifier", expectedString: "b7a16c7a-a718-4ed8-97cb-20ccbadcc339"},
}

func Test_GetRows(t *testing.T) {
	reader := getDbReader(testDbDriver, testDb)
	database, err := reader.ReadSchema()
	if err != nil {
		t.Fatal(err)
	}

	table := findTable("DataTypeTest", database, t)

	// read the data from it
	rows, err := GetRows(reader, nil, table, 999)
	if err != nil {
		t.Fatal(err)
	}

	// check the column count is as expected
	found, countIndex := table.FindCol("colCount")
	if !found {
		t.Fatal("colCount column missing from " + table.String())
	}
	expectedColCount := int(rows[0][countIndex].(int64))
	actualColCount := len(table.Columns)
	if actualColCount != expectedColCount {
		t.Errorf("Expected %#v columns, found %#v", expectedColCount, actualColCount)
	}

	for _, test := range tests {
		if test.row+1 > len(rows) {
			t.Errorf("Not enough rows. %+v", test)
			continue
		}
		found, columnIndex := table.FindCol(test.colName)
		if !found {
			t.Logf("Skipped test for non-existent column %+v", test)
			continue
		}

		actualType := table.Columns[columnIndex].Type
		if !strings.EqualFold(actualType, test.expectedType) {
			t.Errorf("Incorrect column type %s %+v", actualType, test)
		}
		actualString := DbValueToString(rows[test.row][columnIndex], actualType)
		if *actualString != test.expectedString {
			t.Errorf("Incorrect string '%s' %+v", *actualString, test)
		}
	}
}

// find a table. Automatically adds dbo for schema if supported
func findTable(tableName string, database schema.Database, t *testing.T) *schema.Table {
	var schemaName string
	if database.Supports.Schema {
		schemaName = "dbo"
	}
	tableToFind := schema.Table{Schema: schemaName, Name: tableName}
	table := database.FindTable(&tableToFind)
	if table == nil {
		t.Fatal(tableToFind.String() + " table missing")
	}
	return table
}
