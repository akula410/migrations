package src



type config struct{
	fileReport string
	filePrefix string
	fileGenerate string

	dirMigrations string
	dirReport string
	dirGenerate string

	afterInit func()

	methodUp string
	methodDown string
	methodCreate string
	methodInit string

	defaultStep int

	flagMethod string
	flagStep string
	flagTask string

	migrationList []MigrationInterface

	tmplMigration string
	tmplMigrationList string
	tmplFileGenerate string
}

var Config config

func init(){
	Config.fileReport = "report.local.conf"
	Config.filePrefix = "Migration"
	Config.fileGenerate = "MigrateList"

	Config.dirMigrations = "./migrations/"
	Config.dirReport = "./report/"
	Config.dirGenerate = "./generate/"

	Config.methodUp = "up"
	Config.methodDown = "down"
	Config.methodCreate = "create"
	Config.methodInit = "init"

	Config.defaultStep = 0

	Config.flagMethod = "method"
	Config.flagStep = "step"
	Config.flagTask = "task"

	Config.tmplMigration = `package migrations

type {{ .StructureName}} struct{

}

var {{ .PropertyName}} {{.StructureName}}

func (m {{ .StructureName}}) Up(){

}

func (m {{ .StructureName}}) Down(){

}

func (m {{ .StructureName}}) GetName() string{
    return "{{ .PropertyName}}"
}`

Config.tmplMigrationList = `package generate

import (
    "github.com/akula410/migrations"
    {{ .MigrationPackage}}
)

var MigrateList = []migrations.MigrationInterface{
{{ .Migrations}}
}`

	Config.tmplFileGenerate = `package generate
import "github.com/akula410/migrations/src"
var MigrateList []src.MigrateInterface`
}


func (c *config) SetFileReport(way string)*config{
	c.fileReport = way
	return c
}

func (c *config) SetFilePrefix(way string)*config{
	c.filePrefix = way
	return c
}

func (c *config) SetDirMigrations(way string)*config{
	c.dirMigrations = way
	return c
}

func (c *config) SetDirReport(way string)*config{
	c.dirReport = way
	return c
}




func (c *config) SetAfterInit(f func())*config{
	c.afterInit = f
	return c
}

func (c *config) GetDefaultStep()int{
	return c.defaultStep
}

func (c *config) GetFlagMethod()string{
	return c.flagMethod
}

func (c *config) GetFlagStep()string{
	return c.flagStep
}

func (c *config) GetFlagTask()string{
	return c.flagTask
}

func (c *config) GetMethodUp()string{
	return c.methodUp
}

func (c *config) GetMethodDown()string{
	return c.methodDown
}

func (c *config) GetMethodCreate()string{
	return c.methodCreate
}

func (c *config) GetMethodInit()string{
	return c.methodInit
}

func (c *config) GetFileReport()string{
	return c.fileReport
}

func (c *config) GetDirMigrations()string{
	return c.dirMigrations
}
func (c *config) GetDirReport()string{
	return c.dirReport
}
func (c *config) GetDirGenerate()string{
	return c.dirGenerate
}

func (c *config) GetFilePrefix()string{
	return c.filePrefix
}

func (c *config) GetFileGenerate()string{
	return c.fileGenerate
}

func (c *config) SetMigrationList(list []MigrationInterface)string{
	c.migrationList = list
	return c.filePrefix
}

func (c *config) GetMigrationList() []MigrationInterface{
	return c.migrationList
}

func (c *config) GetMigration(i int) MigrationInterface{
	return c.migrationList[i]
}

func (c *config) GetTmplMigration() string{
	return c.tmplMigration
}

func (c *config) GetTmplMigrationList() string{
	return c.tmplMigrationList
}

func (c *config) GetTmplFileGenerate() string{
	return c.tmplFileGenerate
}