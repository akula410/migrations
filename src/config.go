package src



type ConfigAbstract struct{
	fileReport string
	filePrefix string
	fileGenerate string

	dirMigrations string
	dirReport string
	dirGenerate string

	beforeInit func()
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

	packageFileMigration string
}

var Config ConfigAbstract

func init(){
	Config.fileReport = "report.local.conf"
	Config.filePrefix = "Migration"
	Config.fileGenerate = "MigrateList.go"

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
    "github.com/akula410/migrations/src"
    "{{ .MigrationPackage}}"
)

var MigrateList = []src.MigrationInterface{
{{ .Migrations}}
}`

	Config.tmplFileGenerate = `package generate
import "github.com/akula410/migrations/src"
var MigrateList []src.MigrationInterface`
}


func (c *ConfigAbstract) SetFileReport(way string)*ConfigAbstract{
	c.fileReport = way
	return c
}

func (c *ConfigAbstract) SetFilePrefix(way string)*ConfigAbstract{
	c.filePrefix = way
	return c
}

func (c *ConfigAbstract) SetDirMigrations(way string)*ConfigAbstract{
	c.dirMigrations = way
	return c
}

func (c *ConfigAbstract) SetDirReport(way string)*ConfigAbstract{
	c.dirReport = way
	return c
}



func (c *ConfigAbstract) SetBeforeInit(f func())*ConfigAbstract{
	c.beforeInit = f
	return c
}

func (c *ConfigAbstract) SetAfterInit(f func())*ConfigAbstract{
	c.afterInit = f
	return c
}

func (c *ConfigAbstract) SetPackageFileMigration(pack string)*ConfigAbstract{
	c.packageFileMigration = pack
	return c
}





func (c *ConfigAbstract) GetDefaultStep()int{
	return c.defaultStep
}

func (c *ConfigAbstract) GetFlagMethod()string{
	return c.flagMethod
}

func (c *ConfigAbstract) GetFlagStep()string{
	return c.flagStep
}

func (c *ConfigAbstract) GetFlagTask()string{
	return c.flagTask
}

func (c *ConfigAbstract) GetMethodUp()string{
	return c.methodUp
}

func (c *ConfigAbstract) GetMethodDown()string{
	return c.methodDown
}

func (c *ConfigAbstract) GetMethodCreate()string{
	return c.methodCreate
}

func (c *ConfigAbstract) GetMethodInit()string{
	return c.methodInit
}

func (c *ConfigAbstract) GetFileReport()string{
	return c.fileReport
}

func (c *ConfigAbstract) GetDirMigrations()string{
	return c.dirMigrations
}
func (c *ConfigAbstract) GetDirReport()string{
	return c.dirReport
}

func (c *ConfigAbstract) GetBeforeInit() func(){
	return c.beforeInit
}

func (c *ConfigAbstract) GetAfterInit() func(){
	return c.afterInit
}

func (c *ConfigAbstract) GetDirGenerate()string{
	return c.dirGenerate
}

func (c *ConfigAbstract) GetFilePrefix()string{
	return c.filePrefix
}

func (c *ConfigAbstract) GetFileGenerate()string{
	return c.fileGenerate
}

func (c *ConfigAbstract) SetMigrationList(list []MigrationInterface)string{
	c.migrationList = list
	return c.filePrefix
}

func (c *ConfigAbstract) GetMigrationList() []MigrationInterface{
	return c.migrationList
}

func (c *ConfigAbstract) GetMigration(i int) MigrationInterface{
	return c.migrationList[i]
}

func (c *ConfigAbstract) GetTmplMigration() string{
	return c.tmplMigration
}

func (c *ConfigAbstract) GetTmplMigrationList() string{
	return c.tmplMigrationList
}

func (c *ConfigAbstract) GetTmplFileGenerate() string{
	return c.tmplFileGenerate
}

func (c *ConfigAbstract) GetPackageFileMigration() string{
	return c.packageFileMigration
}