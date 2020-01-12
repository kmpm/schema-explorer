package implicitfk

import (
	"fmt"
	"log"
	"sort"

	"github.com/timabell/schema-explorer/schema"
)

type tablePair struct {
	name   string
	cols   []string
	quotas []int
	tables schema.TableList
}

type colTableEntry struct {
	colName string
	tables  schema.TableList
}

type colTableMap map[string]colTableEntry
type pairMap map[string]tablePair

func colKeyGen(col *schema.Column) string {
	if col.Name == "id" {
		return ""
	}
	return fmt.Sprintf("%s|%s", col.Name, col.Type)
}

func createColTabledMap(database *schema.Database, ctmap colTableMap) {
	for _, table := range database.Tables {
		for _, col := range table.Columns {
			if key := colKeyGen(col); key != "" {
				if val, ok := ctmap[key]; ok {
					val.tables = append(val.tables, table)
					ctmap[key] = val
					// log.Printf("found existing col %s. now %d", col.Name, len(cols[col.Name]))
				} else {
					ctmap[key] = colTableEntry{
						colName: col.Name,
						tables:  schema.TableList{table}}
				}
			}
		}
	}
}

func createPairMap(cols colTableMap, pairs pairMap) {
	for key, entry := range cols {
		tableCount := len(entry.tables)
		colName := entry.colName
		if tableCount > 1 {
			sort.Sort(entry.tables)
			log.Printf("candidate implicit key for %s in tables %s", key, entry.tables)
			// Build table pairs

			for i, t1 := range entry.tables {
				if i < tableCount-1 {
					for _, t2 := range entry.tables[i+1:] {
						pair := fmt.Sprintf("%s-%s", t1.Name, t2.Name)
						list := schema.TableList{t1, t2}
						if val, ok := pairs[pair]; ok {
							val.cols = append(val.cols, colName)
							pairs[pair] = val
						} else {
							pairs[pair] = tablePair{name: pair, cols: []string{colName}, tables: list}
						}
					}
				}
			}
		}
	}
}

// calculateQuotas checks the relationship between number of columns in the primary key
// and how many of the "found" columns is in the primary key
// 100 = the column(s) is same as PK.
func calculateQuotas(p *tablePair) {
	// Check primary key count for fields
	// alone in PK is more worth then part of composite
	sort.Strings(p.cols)
	p.quotas = []int{0, 0}
	for ti, table := range p.tables {
		pkSize := 0
		inPrimaryKey := 0
		for _, col := range table.Columns {
			if col.IsInPrimaryKey {
				pkSize++
				if _, ok := isStringIn(col.Name, p.cols); ok {
					// log.Printf("Col: %s, Pos: %d in %v", col.Name, pos, p.cols)
					inPrimaryKey++
				}
			}
		}
		p.quotas[ti] = 0
		if pkSize > 0 {
			p.quotas[ti] = inPrimaryKey * 10 / pkSize * 10
		}
	}
}

// getFkTables will return (source, dest, destQuota)
func (pair *tablePair) getFkTables() (*schema.Table, *schema.Table, int) {
	calculateQuotas(pair)
	// the one with the lowest q should be source
	src := 0
	dst := 1
	if pair.quotas[0] > pair.quotas[1] {
		src = 1
		dst = 0
	}
	return pair.tables[src], pair.tables[dst], pair.quotas[dst]
}

func isStringIn(target string, list []string) (pos int, ok bool) {
	pos = sort.SearchStrings(list, target)
	ok = pos < len(list) && list[pos] == target
	return
}

func implicitfkByName(database *schema.Database) {
	// 1. Build a list of field names from all tables
	cols := make(colTableMap)
	createColTabledMap(database, cols)

	// 2. Check if any field exists in more than 1 table
	// and 3. Get ALL common fields for possible composite keys
	pairs := make(pairMap)
	createPairMap(cols, pairs)

	for _, p := range pairs {
		sourceTable, destTable, destQuota := p.getFkTables()
		if destQuota == 0 {
			// if the best option isn't a part of PK then skip altogether
			continue
		}
		scols, _ := sourceTable.FindColumns(p.cols...)
		dcols, _ := destTable.FindColumns(p.cols...)
		found := false
		// check for existing FKs
		for _, fk := range database.Fks {
			if fk.SourceTable != sourceTable || fk.DestinationTable != destTable {
				continue
			}
			if !fk.SourceColumns.Equals(scols) || !fk.DestinationColumns.Equals(dcols) {
				continue
			}
			found = true
			log.Printf("Fk exists for %s(%s) to %s(%s). Skip syntethic", sourceTable, scols, destTable, dcols)
		}
		// None were found
		if !found {
			log.Printf("Fk created for %s(%s) to %s(%s). Create syntethic", sourceTable, scols, destTable, dcols)
			fk := &schema.Fk{
				SourceTable:        sourceTable,
				DestinationTable:   destTable,
				SourceColumns:      scols,
				DestinationColumns: dcols,
				Implicit:           true,
			}
			database.AddFk(fk)
		}
	}
}
