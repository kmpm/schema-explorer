package schema

import "testing"

func TestIndex_String(t *testing.T) {
	t.Skip("Needs test cases")
	tests := []struct {
		name  string
		index Index
		want  string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.index.String(); got != tt.want {
				t.Errorf("Index.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_String(t *testing.T) {
	t.Skip("Needs test cases")
	tests := []struct {
		name  string
		table Table
		want  string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.table.String(); got != tt.want {
				t.Errorf("Table.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTableList_String(t *testing.T) {
	t1 := &Table{Name: "Alpha"}
	t2 := &Table{Name: "Beta"}
	t3 := &Table{Name: "Gamma"}
	tl1 := TableList{t3, t1, t2}
	tests := []struct {
		name   string
		tables TableList
		want   string
	}{
		{tables: tl1, want: "Gamma,Alpha,Beta"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tables.String(); got != tt.want {
				t.Errorf("TableList.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColumnList_String(t *testing.T) {
	c1 := &Column{Name: "Alpha"}
	c2 := &Column{Name: "Beta"}
	c3 := &Column{Name: "Gamma"}
	cl1 := ColumnList{c3, c1, c2}
	tests := []struct {
		name    string
		columns ColumnList
		want    string
	}{
		{columns: cl1, want: "Gamma,Alpha,Beta"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.columns.String(); got != tt.want {
				t.Errorf("ColumnList.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColumn_String(t *testing.T) {
	t.Skip("Needs test cases")
	tests := []struct {
		name   string
		column Column
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.column.String(); got != tt.want {
				t.Errorf("Column.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFk_String(t *testing.T) {
	t.Skip("Needs test cases")
	tests := []struct {
		name string
		fk   Fk
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fk.String(); got != tt.want {
				t.Errorf("Fk.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
