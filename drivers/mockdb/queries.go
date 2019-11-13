// +build mock

package mockdb

import sqlmock "github.com/DATA-DOG/go-sqlmock"

func expectSelectMockMaster(mock sqlmock.Sqlmock) {
	rows := sqlmock.NewRows([]string{"name", "type"}).
		AddRow("DataTypeTest", "table").
		AddRow("toy", "table").
		AddRow("person", "table").
		AddRow("pet", "table").
		AddRow("SortFilterTest", "table")

	mock.ExpectQuery("SELECT name FROM mock_master WHERE type='table' AND name not like 'mock_%' order by name;").WillReturnRows(rows)
}
