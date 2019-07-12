package migrations

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"migrations/src"
	"os"
	"strings"
)

type Management struct {

}

func Init()*Management{
	Method := flag.String(src.Config.GetFlagMethod(), "", "a string")
	Step := flag.Int(src.Config.GetFlagStep(), src.Config.GetDefaultStep(), "a int")
	Task := flag.String(src.Config.GetFlagTask(), "", "a string")
	flag.Parse()
	var r = &Management{}

	switch *Method{
	case src.Config.GetMethodUp():
		r.ApplyUp(Step)
	case src.Config.GetMethodDown():
		r.ApplyDown(Step)
	case src.Config.GetMethodCreate():
		r.CreateMigration(Task)
	case src.Config.GetMethodInit():
		r.CreateStructure()
	}
	return r
}

func (r *Management) ApplyUp(step *int){
	counter := 0
	for i := len(src.Config.GetMigrationList())-1; i >= 0; i-- {
		if !r.getResult(src.Config.GetMigration(i).GetName()) {
			if counter == *step && *step != 0 {
				break
			}
			src.Config.GetMigration(i).Up()
			r.setResult(src.Config.GetMigration(i).GetName(), true)
			fmt.Println(fmt.Sprintf("Migration %s up", src.Config.GetMigration(i).GetName()))
			counter++
		}
	}
}

func (r *Management) ApplyDown(step *int){
	if *step == 0 {
		*step = 1
	}
	counter := 0
	for _, m := range src.Config.GetMigrationList() {
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

func (r *Management) CreateMigration(task *string){

	name := fmt.Sprintf("%s%s%s", src.Config.GetFilePrefix(), src.UUID.GetUUID(), *task)
	r.createMigrationFile(name).setMigrationReport(name).syncFileReportInMigrateList()
}

func findDir(){
	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Println(file.Name(), file.IsDir())
	}
}

func (r *Management) createMigrationFile(name string) *Management{
	findDir()
	tmpl, err := template.ParseFiles("/templates/Migration.tmpl")
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

	err = ioutil.WriteFile(fmt.Sprintf("%s%s.go", src.Config.GetDirMigrations(), name), []byte(tpl.String()), 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println(fmt.Sprintf("Migration %s has been create", name))
	return r
}

func (r *Management) syncFileReportInMigrateList(){
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

func (r *Management) CreateStructure(){

}

func (r *Management) setMigrationReport(name string)*Management{
	var err error
	var newFile []string

	//Добавить новую миграцию в начало файла
	newFile = append(newFile, fmt.Sprintf("%s %s", name, "false"))
	var scanFunc = func(scanText string){
		if scanText = strings.Trim(scanText, "\r\n "); len(scanText)>0 {
			newFile = append(newFile, scanText)
		}
	}

	r.scanReportFile(src.Config.GetFileReport(), scanFunc)
	// обновить данные во всем файле
	err = ioutil.WriteFile(src.Config.GetFileReport(), []byte(strings.Join(newFile, "\r\n")), 0644)
	if err != nil {
		panic(err)
	}
	return r
}

func (r *Management) askReportFile()bool{
	var err error

	result := true

	_, err = os.Stat(src.Config.GetFileReport())
	if err != nil {
		result = false
	}
	return result
}


func (r *Management) createReportFile()*Management{
	var err error
	var file *os.File
	file, err = os.Create(src.Config.GetFileReport())
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
func (r *Management) getResult(name string) bool{
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

	r.scanReportFile(src.Config.GetFileReport(), scanFunc)
	return result
}

//Изменение состояния записи в отчете файла
func (r *Management) setResult(name string, result bool){

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

	r.scanReportFile(src.Config.GetFileReport(), scanFunc)

	// write the whole body at once
	err = ioutil.WriteFile(src.Config.GetFileReport(), []byte(strings.Join(newFile, "\r\n")), 0644)
	if err != nil {
		panic(err)
	}
}

//Работа с файлом отчета миграций, построчно (scanFunc)
func (r *Management) scanReportFile(way string, scanFunc func(string)){
	var file *os.File
	var err error

	file, err = os.Open(src.Config.GetFileReport())
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

func (r *Management) getMigrationNames() []string {
	var result = make([]string, 0)
	var scanFunc = func(scanText string){
		transformText := strings.Split(scanText, " ")
		if len(transformText) == 2 {
			result = append(result, transformText[0])
		}
	}
	r.scanReportFile(src.Config.GetFileReport(), scanFunc)
	return result
}
