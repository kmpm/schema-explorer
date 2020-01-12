package implicitfk

import (
	"strings"
	"testing"

	"github.com/timabell/schema-explorer/schema"
)

func newFakeTable(tableName string, colNames ...string) (table *schema.Table) {
	table = &schema.Table{Name: tableName}
	columns := make(schema.ColumnList, len(colNames))
	table.Columns = columns
	for index, colName := range colNames {
		cn := strings.ReplaceAll(colName, "*", "")
		col := &schema.Column{Name: cn, Type: "string"}
		col.IsInPrimaryKey = strings.HasSuffix(colName, "*")
		columns[index] = col
	}
	return
}

func newFakeDb() *schema.Database {
	db := &schema.Database{}
	db.Tables = schema.TableList{
		newFakeTable("Alpha", "someId*", "something"),
		newFakeTable("Beta", "betaId*", "someId*", "somethingelse"),
		newFakeTable("Gamma", "id*", "someId", "betaId", "completely_different"),
	}

	return db
}

func Test_implicitfkByName(t *testing.T) {
	type args struct {
		database *schema.Database
	}
	tests := []struct {
		name    string
		args    args
		wantFks int
	}{
		{args: args{database: newFakeDb()}, wantFks: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			implicitfkByName(tt.args.database)
			if len(tt.args.database.Fks) != tt.wantFks {
				t.Errorf("Want %d Fks in database but got %d", tt.wantFks, len(tt.args.database.Fks))
			}
		})
	}
}
