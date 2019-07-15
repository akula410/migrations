package main

import (
	"github.com/akula410/migrations"
	"migrations/src"

	//"migrations/example/generate"
)

func main(){
	migrations/src.Config.SetPackageFileMigration("migrations/example/generate")
	migrations.Init()
}
