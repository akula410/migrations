package main

import (
	"github.com/akula410/migrations"
	c "github.com/akula410/migrations/src"
	//"migrations/example/generate"
)

/**
	example -method=init
	uncomment
		"migrations/example/generate"
		config.SetMigrationList(generate.MigrateList)
	go build
	example -method=create
	go build
	example -method=up
	example -method=down -step=1

 */
func main(){
	config := c.Config
	config.SetPackageFileMigration("migrations/example/migrations")
	//config.SetMigrationList(generate.MigrateList)

	Management := &migrations.Management{}
	Management.SetConfig(config).Init()
}
