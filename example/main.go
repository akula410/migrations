package main

import (
	"github.com/akula410/migrations"
	c "github.com/akula410/migrations/src"

	//"migrations/example/generate"
)

func main(){
	config := c.Config

	c.Config.SetPackageFileMigration("migrations/example/generate")
	migrations.Init()
}
