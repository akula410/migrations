package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type migrateManagement struct {

}

func main(){
	Method := flag.String(Config.GetFlagMethod(), "", "a string")
	Step := flag.Int(Config.GetFlagStep(), Config.GetDefaultStep(), "a int")
	Task := flag.String(Config.GetFlagTask(), "", "a string")
	flag.Parse()
	var r = &migrateManagement{}

	switch *Method{
	case Config.GetMethodUp():
		r.ApplyUp(Step)
	case Config.GetMethodDown():
		r.ApplyDown(Step)
	case Config.GetMethodCreate():
		r.CreateMigration(Task)
	case Config.GetMethodInit():
		r.Init()
	}
}

func (r *migrateManagement) ApplyUp(step *int){
	counter := 0
	for i := len(Config.GetMigrationList())-1; i >= 0; i-- {
		if !r.getResult(Config.GetMigration(i).GetName()) {
			if counter == *step && *step != 0 {
				break
			}
			Config.GetMigration(i).Up()
			r.setResult(Config.GetMigration(i).GetName(), true)
			fmt.Println(fmt.Sprintf("Migration %s up", Config.GetMigration(i).GetName()))
			counter++
		}
	}
}

func (r *migrateManagement) ApplyDown(step *int){
	if *step == 0 {
		*step = 1
	}
	counter := 0
	for _, m := range Config.GetMigrationList() {
		if r.getResult(m.GetName()) {
			if counter == *step {
				break
			}
			m.Down()
			r.setResult(m.GetName(), false)
			fmt.Println(fmt.Sprintf("Migration %s down", m.GetName()))
			counter++
		}
	}
}

func (r *migrateManagement) CreateMigration(task *string){

	name := fmt.Sprintf("%s%s%s", Config.GetFilePrefix(), UUID.GetUUID(), *task)
	r.createMigrationFile(name).setMigrationReport(name).syncFileReportInMigrateList()
}

func (r *migrateManagement) createMigrationFile(name string) *migrateManagement{
	tmpl, err := template.ParseFiles("templates/Migration.tmpl")
	if err != nil {
		panic(err)
	}

	data := struct {
		StructureName string
		PropertyName string
	}{
		StructureName:strings.ToLower(name),
		PropertyName:name,
	}

	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(fmt.Sprintf("%s%s.go", Config.GetDirMigrations(), name), []byte(tpl.String()), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println(fmt.Sprintf("Migration %s has been create", name))
	return r
}

func (r *migrateManagement) syncFileReportInMigrateList(){
	var methods = make([]string, 0)
	//var err error

	for _, n := range r.getMigrationNames() {
		methods = append(methods, fmt.Sprintf("	migrations.%s", n))
	}

	tmpl, err := template.ParseFiles("templates/MigrationList.tmpl")
	if err != nil {
		panic(err)
	}


	data := struct{
		Migrations string
	}{Migrations: fmt.Sprintf("%s,", strings.Join(methods, ",\r\n"))}


	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile("generate/MigrationList.go", []byte(tpl.String()), 0644)
	if err != nil {
		panic(err)
	}
}

func (r *migrateManagement) Init(){

}

func (r *migrateManagement) setMigrationReport(name string)*migrateManagement{
	var err error
	var newFile []string

	//Добавить новую миграцию в начало файла
	newFile = append(newFile, fmt.Sprintf("%s %s", name, "false"))
	var scanFunc = func(scanText string){
		if scanText = strings.Trim(scanText, "\r\n "); len(scanText)>0 {
			newFile = append(newFile, scanText)
		}
	}

	r.scanReportFile(Config.GetFileReport(), scanFunc)
	// обновить данные во всем файле
	err = ioutil.WriteFile(Config.GetFileReport(), []byte(strings.Join(newFile, "\r\n")), 0644)
	if err != nil {
		panic(err)
	}
	return r
}

func (r *migrateManagement) askReportFile()bool{
	var err error

	result := true

	_, err = os.Stat(Config.GetFileReport())
	if err != nil {
		result = false
	}
	return result
}


func (r *migrateManagement) createReportFile()*migrateManagement{
	var err error
	var file *os.File
	file, err = os.Create(Config.GetFileReport())
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
	return r
}


//Проверка состояния записи в отчете файла
func (r *migrateManagement) getResult(name string) bool{
	var result bool

	var scanFunc = func(scanText string){
		if len(scanText)>0 {
			resultString := strings.Split(scanText, " ")
			if len(resultString)!=2 {
				panic("Error config file")
			}
			if name==resultString[0]{
				switch resultString[1] {
				case "true":
					result=true
				case "false":
					result=false
				default:
					panic("Error config result")
				}
			}
		}
	}

	r.scanReportFile(Config.GetFileReport(), scanFunc)
	return result
}

//Изменение состояния записи в отчете файла
func (r *migrateManagement) setResult(name string, result bool){

	var err error
	var resultString []string
	var newFile []string

	var scanFunc = func(scanText string){
		if len(scanText)>0 {
			resultString = strings.Split(scanText, " ")
			if len(resultString)!=2 {
				panic("Error config file")
			}
			if name==resultString[0]{
				var textResult string
				switch result {
				case true:
					textResult = "true"
				case false:
					textResult = "false"
				}

				scanText = fmt.Sprintf("%s %s", resultString[0], textResult)
			}
		}
		newFile = append(newFile, strings.Trim(scanText, "\r\n "))
	}

	r.scanReportFile(Config.GetFileReport(), scanFunc)

	// write the whole body at once
	err = ioutil.WriteFile(Config.GetFileReport(), []byte(strings.Join(newFile, "\r\n")), 0644)
	if err != nil {
		panic(err)
	}
}

//Работа с файлом отчета миграций, построчно (scanFunc)
func (r *migrateManagement) scanReportFile(way string, scanFunc func(string)){
	var file *os.File
	var err error

	file, err = os.Open(Config.GetFileReport())
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(file)


	for scanner.Scan() {
		scanFunc(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	err = file.Close()

	if err != nil {
		panic(err)
	}
}

func (r *migrateManagement) getMigrationNames() []string {
	var result = make([]string, 0)
	var scanFunc = func(scanText string){
		transformText := strings.Split(scanText, " ")
		if len(transformText) == 2 {
			result = append(result, transformText[0])
		}
	}
	r.scanReportFile(Config.GetFileReport(), scanFunc)
	return result
}
