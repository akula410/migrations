package main

import (
	"github.com/akula410/migrations"
	c "github.com/akula410/migrations/src"
	//"migrations/example/generate"
)

/**
	example -m=i
	uncomment
		"migrations/example/generate"
		config.SetMigrationList(generate.MigrateList)
	go build
	example -m=c
	go build
	example -m=u
	example -m=d -s=1

 */
func main(){
	config := c.Config
	config.SetPackageFileMigration("migrations/example/migrations")
	//config.SetMigrationList(generate.MigrateList)

	Management := &migrations.Management{}
	Management.SetConfig(config).Init()
}
