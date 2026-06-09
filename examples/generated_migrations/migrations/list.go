package migrations

import base "github.com/akula410/migrations/v2"

// List contains all registered migrations in version order.
var List = []base.Migration{
	CreateUsersTable20260608120000{},
}
