package migrations

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	c "github.com/akula410/migrations/src"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type Management struct {
	config c.ConfigAbstract
}

func (r *Management) Init()*Management{
	Method := flag.String(r.config.GetFlagMethod(), "", "a string")
	Step := flag.Int(r.config.GetFlagStep(), r.config.GetDefaultStep(), "a int")
	Task := flag.String(r.config.GetFlagTask(), "", "a string")
	flag.Parse()

	switch *Method{
	case r.config.GetMethodUp():
		r.syncMigrateListInFileReport().ApplyUp(Step)
	case r.config.GetMethodDown():
		r.syncMigrateListInFileReport().ApplyDown(Step)
	case r.config.GetMethodCreate():
		r.CreateMigration(Task)
	case r.config.GetMethodInit():
		r.CreateStructure()
	}
	return r
}

func (r *Management) ApplyUp(step *int){
	counter := 0
	for i := len(r.config.GetMigrationList())-1; i >= 0; i-- {
		if res, _ := r.getResult(r.config.GetMigration(i).GetName()); res==false {
			if counter == *step && *step != 0 {
				break
			}
			r.config.GetMigration(i).Up()
			r.setResult(r.config.GetMigration(i).GetName(), true)
			fmt.Println(fmt.Sprintf("Migration %s up", r.config.GetMigration(i).GetName()))
			counter++
		}
	}
}

func (r *Management) ApplyDown(step *int){
	if *step == 0 {
		*step = 1
	}
	counter := 0
	for _, m := range r.config.GetMigrationList() {
		if res, _ := r.getResult(m.GetName()); res==true {
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
	name := fmt.Sprintf("%s%s%s", r.config.GetFilePrefix(), c.UUID.GetUUID(), *task)
	r.createMigrationFile(name).createMigrationReport().setMigrationReport(name).syncFileReportInMigrateList()
}


func (r *Management) createMigrationFile(name string) *Management{
	t := template.New("Migration")

	tmpl, err := t.Parse(r.config.GetTmplMigration())
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

	fmt.Println()

	err = ioutil.WriteFile(fmt.Sprintf("%s%s.go", r.config.GetDirMigrations(), name), []byte(tpl.String()), 0644)
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

	t := template.New("MigrationList")

	tmpl, err := t.Parse(r.config.GetTmplMigrationList())
	if err != nil {
		panic(err)
	}


	data := struct{
		Migrations string
		MigrationPackage string
	}{Migrations: fmt.Sprintf("%s,", strings.Join(methods, ",\r\n")), MigrationPackage: r.config.GetPackageFileMigration()}


	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(fmt.Sprintf("%s%s", r.config.GetDirGenerate(), r.config.GetFileGenerate()), []byte(tpl.String()), 0644)
	if err != nil {
		panic(err)
	}
}

//При переносе данных между ветками, файл конфигурации не переносится
func (r *Management) syncMigrateListInFileReport()*Management{
	r.createDirReport().createMigrationReport()

	for _, m := range r.config.GetMigrationList() {
		if _, ok := r.getResult(m.GetName()); ok==false {
			r.setMigrationReport(m.GetName())
		}
	}

	return r
}

func (r *Management) CreateStructure(){
	if f := r.config.GetBeforeInit(); f != nil {
		f()
	}
	r.createDirMigration().createDirReport().createDirGenerate().createScriptMigrationList()
	if f := r.config.GetAfterInit(); f != nil {
		f()
	}
}

func (r *Management) SetConfig(config c.ConfigAbstract) *Management {
	r.config = config
	return r
}

func (r *Management) GetConfig() c.ConfigAbstract {
	return r.config
}

func (r *Management) createDirMigration()*Management{
	r.createDir(r.config.GetDirMigrations(), 0644)
	return r
}

func (r *Management) createDirReport()*Management{
	r.createDir(r.config.GetDirReport(), 0644)
	return r
}

func (r *Management) createDirGenerate()*Management{
	r.createDir(r.config.GetDirGenerate(), 0644)
	return r
}

func (r *Management) createScriptMigrationList(){
	var err error

	_, err = os.Stat(fmt.Sprintf("%s%s", r.config.GetDirGenerate(), r.config.GetFileGenerate()))
	if err != nil {
		var file *os.File
		file, err = os.Create(fmt.Sprintf("%s%s", r.config.GetDirGenerate(), r.config.GetFileGenerate()))

		if err != nil {
			panic(err)
		}
		err = file.Close()


		if err != nil {
			panic(err)
		}

		// обновить данные во всем файле
		err = ioutil.WriteFile(fmt.Sprintf("%s%s", r.config.GetDirGenerate(), r.config.GetFileGenerate()), []byte(r.config.GetTmplFileGenerate()), 0644)
		if err != nil {
			panic(err)
		}
	}

}

func (r *Management) createDir(path string, mode os.FileMode){
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.Mkdir(path, mode)
		if err != nil {
			panic(err)
		}
	}
}

func (r *Management) setMigrationReport(name string)*Management{
	var err error
	var newFile = make([]string, 0)

	//Добавить новую миграцию в начало файла
	newFile = append(newFile, fmt.Sprintf("%s %s", name, "false"))
	var scanFunc = func(scanText string){
		if scanText = strings.Trim(scanText, "\r\n "); len(scanText)>0 {
			newFile = append(newFile, scanText)
		}
	}

	r.scanReportFile(fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport()), scanFunc)
	// обновить данные во всем файле
	err = ioutil.WriteFile(fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport()), []byte(strings.Join(newFile, "\r\n")), 0644)
	if err != nil {
		panic(err)
	}
	return r
}

func (r *Management) createMigrationReport()*Management{
	var err error
	var file *os.File
	path := fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport())

	if r.askReportFile() == false{
		file, err = os.Create(path)
		if err != nil {
			panic(err)
		}
		err = file.Close()
		if err != nil {
			panic(err)
		}
	}

	return r
}

func (r *Management) askReportFile()bool{
	var err error
	path := fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport())
	result := true

	_, err = os.Stat(path)
	if err != nil {
		result = false
	}
	return result
}


func (r *Management) createReportFile()*Management{
	var err error
	var file *os.File
	file, err = os.Create(fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport()))
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
func (r *Management) getResult(name string) (bool, bool){
	var result bool
	var ok bool

	ok = true

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
					ok = false
				}
			}
		}
	}

	r.scanReportFile(fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport()), scanFunc)
	return result, ok
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

	r.scanReportFile(fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport()), scanFunc)

	// write the whole body at once
	err = ioutil.WriteFile(fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport()), []byte(strings.Join(newFile, "\r\n")), 0644)
	if err != nil {
		panic(err)
	}
}

//Работа с файлом отчета миграций, построчно (scanFunc)
func (r *Management) scanReportFile(way string, scanFunc func(string)){
	var file *os.File
	var err error

	file, err = os.Open(fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport()))
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
	r.scanReportFile(fmt.Sprintf("%s%s", r.config.GetDirReport(), r.config.GetFileReport()), scanFunc)
	return result
}
