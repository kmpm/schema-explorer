package schema

import (
	"fmt"
	"strings"
)

type SupportedFeatures struct {
	Schema               bool
	Descriptions         bool
	FkNames              bool
	PagingWithoutSorting bool
}

type Database struct {
	Name              string // if available, used for url building
	Tables            []*Table
	Fks               []*Fk
	Indexes           []*Index
	Supports          SupportedFeatures
	Description       string
	DefaultSchemaName string
}

type Pk struct {
	Name    string
	Columns ColumnList
}

type Index struct {
	Name        string
	Columns     ColumnList
	IsUnique    bool
	IsClustered bool
	IsDisabled  bool
	Table       *Table
}

func (index Index) String() string {
	unique := ""
	if index.IsUnique {
		unique = "Unique "
	}
	return fmt.Sprintf("%sIndex %s on %s(%s)", unique, index.Name, index.Table.String(), index.Columns.String())
}

type Table struct {
	Schema      string
	Name        string
	Columns     ColumnList
	Pk          *Pk
	Fks         []*Fk
	InboundFks  []*Fk
	Indexes     []*Index
	Description string
	RowCount    *int       // pointer to allow us to tell the difference between zero and unknown
	PeekColumns ColumnList // list of columns to show as a preview when this is a target for a join, e.g. the "Name" column. The schema readers are not expected to populate this field.
}

type TableList []*Table

// implement sort.Interface for list of tables https://stackoverflow.com/a/19948360/10245
func (tables TableList) Len() int {
	return len(tables)
}
func (tables TableList) Swap(i, j int) {
	tables[i], tables[j] = tables[j], tables[i]
}
func (tables TableList) Less(i, j int) bool {
	return tables[i].String() < tables[j].String()
}

type ColumnList []*Column

type Column struct {
	Position       int
	Name           string
	Type           string
	Fks            []*Fk
	InboundFks     []*Fk
	Indexes        []*Index
	Description    string
	IsInPrimaryKey bool
	Nullable       bool
}

// Fk represents a foreign key
type Fk struct {
	Id                 int
	Name               string
	SourceTable        *Table
	SourceColumns      ColumnList
	DestinationTable   *Table
	DestinationColumns ColumnList
	Implicit           bool
}

// Simplified fk constructor for single-column foreign keys
func NewFk(name string, sourceTable *Table, sourceColumn *Column, destinationTable *Table, destinationColumn *Column) *Fk {
	return &Fk{Name: name, SourceTable: sourceTable, SourceColumns: ColumnList{sourceColumn}, DestinationTable: destinationTable, DestinationColumns: ColumnList{destinationColumn}}
}

func (table Table) String() string {
	if table.Schema == "" {
		return table.Name
	}
	return table.Schema + "." + table.Name
}

// reconstructs schema+name from "schema.name" string
func TableFromString(value string) Table {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) == 2 {
		return Table{Schema: parts[0], Name: parts[1]}
	}
	return Table{Schema: "", Name: parts[0]}
}

func (tables TableList) String() string {
	var tableNames []string
	for _, table := range tables {
		tableNames = append(tableNames, table.Name)
	}
	return strings.Join(tableNames, ",")
}

func (columns ColumnList) String() string {
	var columnNames []string
	for _, col := range columns {
		columnNames = append(columnNames, col.Name)
	}
	return strings.Join(columnNames, ",")
}

// Equals checks if one list contains the same as the other
// irrespective of order
func (columns ColumnList) Equals(other ColumnList) (same bool) {
	// if not same size they can't be equal
	if same = len(columns) == len(other); !same {
		return
	}
	matches := 0
	for _, c1 := range columns {
		for _, c2 := range other {
			if c1 == c2 {
				matches++
				continue
			}
		}
	}
	same = matches == len(columns)
	return
}

func (column Column) String() string {
	return column.Name
}

func (fk Fk) String() string {
	return fmt.Sprintf("%s %s(%s) => %s(%s)", fk.Name, fk.SourceTable, fk.SourceColumns.String(), fk.DestinationTable, fk.DestinationColumns.String())
}

// returns nil if not found.
// searches on schema+name
func (database Database) FindTable(tableToFind *Table) (table *Table) {
	for _, table := range database.Tables {
		if (!database.Supports.Schema || table.Schema == tableToFind.Schema) && table.Name == tableToFind.Name {
			return table
		}
	}
	return nil
}

func (database *Database) AddFk(fk *Fk) {
	database.Fks = append(database.Fks, fk)
	st := database.FindTable(fk.SourceTable)
	// hook up inbound
	st.Fks = append(st.Fks, fk)
	dt := database.FindTable(fk.DestinationTable)
	if dt != nil {
		dt.InboundFks = append(dt.InboundFks, fk)
		for _, destCol := range fk.DestinationColumns {
			destCol.InboundFks = append(destCol.InboundFks, fk)
		}
	}
}

func (table Table) FindColumn(columnName string) (index int, column *Column) {
	for index, col := range table.Columns {
		if col.Name == columnName {
			return index, col
		}
	}
	return -1, nil
}

// FindColumns returns a ColumnList of given columns
func (table Table) FindColumns(colNames ...string) (list ColumnList, err error) {
	for _, colName := range colNames {
		_, col := table.FindColumn(colName)
		if col == nil {
			err = fmt.Errorf("Column '%s' not found in table", colName)
			return
		}
		list = append(list, col)
	}
	return
}
