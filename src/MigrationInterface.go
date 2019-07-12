package src

type MigrationInterface interface {
	Up()
	Down()
	GetName()string
}
