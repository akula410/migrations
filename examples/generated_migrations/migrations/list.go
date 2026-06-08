package migrations

import base "github.com/akula410/migrations"

// List contains all registered migrations in version order.
var List = []base.Migration{
	CreateUsersTable20260608120000{},
}
