package implicitfk

import "github.com/timabell/schema-explorer/schema"

func Generate(mode string, database *schema.Database) {
	if mode == "byname" {
		implicitfkByName(database)
	}
}
