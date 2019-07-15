package main

import (
	"github.com/akula410/migrations"
	c "github.com/akula410/migrations/src"

	//"migrations/example/generate"
)

func main(){
	config := c.Config
	config.SetPackageFileMigration("migrations/example/generate")

	Management := &migrations.Management{}
	Management.SetConfig(config)

	migrations.Init()
}
