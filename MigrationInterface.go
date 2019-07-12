package main

type MigrationInterface interface {
	Up()
	Down()
	GetName()string
}
