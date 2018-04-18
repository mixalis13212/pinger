// main project main.go
package main

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	"github.com/go-ini/ini"
)

var configFileName = "satmon.ini"
var serversFileName = "servers.ini"
var hourNow int64

//NumOfCheck - Number of check one URL
var NumOfCheck = 5

//HTTPTimeOut - timeout for check one URL
var HTTPTimeOut = 10 * time.Second

//TimeOutSleep timeout between check all URL's
var TimeOutSleep = 60 * time.Minute

//ServerAttr - attribute and status of server
type ServerAttr struct {
	IP      string
	Note    string
	SiteID  string
	CodeNow int
	Code    [24]int
}

//Servers - list of servers for check
var servers ServersType

// ServersType type for list of servers for check
type ServersType struct {
	data map[string]ServerAttr
}

func intro() {
	fmt.Printf(`
                          Sattelite Monitor for 
      "Management of Technological Transport & Special Mechanism Burservice"
                             Copyright (C) 2018
SatMon 0.6                       Vlad Vegner                       April 13 2018
================================================================================
`)
}

//check URL
func checkURL(ip string) (int, string) {
	// возвращает true — если сервис доступен, false, если нет и текст сообщения
	url := "http://" + ip + "/"
	fmt.Printf("Проверяем адрес %v ", url)
	//Отключаем проверку сертиикатов
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	//Настраиваем клиент для работы с отключенной проверкой сертификатов и устанавливаем время ответа в 10 секкунд
	client := http.Client{Timeout: HTTPTimeOut, Transport: tr}
	resp, err := client.Get(url)
	if err != nil {
		if strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
			return 1, fmt.Sprintf("Ошибка. Сервер не ответил вовремя")
		}
		return 1, fmt.Sprintf("Ошибка соединения. %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 1, fmt.Sprintf("Ошибка. http-статус: %v", resp.StatusCode)
	}
	return 2, fmt.Sprintf("Онлайн. http-статус: %d", resp.StatusCode)
}

func (s *ServersType) myhandler(w http.ResponseWriter, r *http.Request) {
	// мапа доступна через s.data
	// fmt.Println("Эти данные необходимы для разработчика")
	// fmt.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	// fmt.Printf("Печать из indexHandler\n%#v\n", s.data)
	// fmt.Println("+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	// fmt.Println("выполняется myhandler")
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}
	err = tmpl.Execute(w, s.data)
	if err != nil {
		fmt.Fprint(w, err.Error())
	}
}

func (sA *ServerAttr) checkElement() {
	tm := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("%v ", tm)
	var working int
	var msg string
	var err error
	for i := NumOfCheck; i > 1; i-- {
		working, msg = checkURL(sA.IP)
		if working == 2 {
			break
		}
		fmt.Printf("Не удалось. \n Еще раз...         ")
	}
	fmt.Println(msg)
	hourNow, err = strconv.ParseInt(time.Now().Format("15"), 10, 8)
	if (err != nil) || (hourNow > 23) || (hourNow < 0) {
		fmt.Println("Не удалось определить час", hourNow, err)
		sA.CodeNow = working
	} else {
		sA.Code[hourNow] = working
		sA.CodeNow = working
	}
}

//LoadServers - Load list of Servers for check thier http-status
func (s *ServersType) LoadServers(nameFile string) {
	fmt.Printf("%v Загружаю список серверов из %v", time.Now().Format("15:04:05"), nameFile)
	cfg, err := ini.Load(nameFile)
	if err != nil {
		fmt.Printf("\nОшибка чтения конфигурационного файла: %v", err)
		os.Exit(1)
	}
	fmt.Printf("\t...\tОК\n")
	names := cfg.SectionStrings()
	var serverElm ServerAttr
	s.data = make(map[string]ServerAttr)
	for _, name := range names {
		if name == "DEFAULT" {
			continue
		}
		serverElm.IP = cfg.Section(name).Key("IP").String()
		serverElm.Note = cfg.Section(name).Key("Note").String()
		serverElm.SiteID = cfg.Section(name).Key("SiteID").String()
		s.data[name] = serverElm
	}
}

func (s *ServersType) getreport(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("выполняется getreport")
	outputFileName, err := s.makereport()
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}
	tmpl, err := template.ParseFiles("templates/getreport.html")
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}
	err = tmpl.Execute(w, outputFileName)
	if err != nil {
		fmt.Fprint(w, err.Error())
	}

}

func (s *ServersType) makereport() (string, error) {
	// fmt.Println("выполняется makereport")
	wSheet := "Отчет"
	templateFile := "templates/Template.xlsx"
	index := 1
	outputFileName := time.Now().Format("2006-01-02") + ".xlsx"
	xlsx, err := excelize.OpenFile(templateFile)
	if err != nil {
		fmt.Println(err)
		return outputFileName, err
	}
	for key, value := range s.data {
		xlsx.SetCellValue(wSheet, "B"+strconv.Itoa(index+7), index)
		xlsx.SetCellValue(wSheet, "C"+strconv.Itoa(index+7), value.Note)
		xlsx.SetCellValue(wSheet, "E"+strconv.Itoa(index+7), key)
		xlsx.SetCellValue(wSheet, "F"+strconv.Itoa(index+7), value.SiteID)
		if value.CodeNow == 2 {
			xlsx.SetCellValue(wSheet, "D"+strconv.Itoa(index+7), "В сети")
		} else {
			xlsx.SetCellValue(wSheet, "D"+strconv.Itoa(index+7), "Не в сети")
		}
		index++
	}
	xlsx.SetCellValue(wSheet, "D4", time.Now().Format("02.01.2006"))
	// Save xlsx file by the given path.
	err = xlsx.SaveAs("./report/" + outputFileName)
	if err != nil {
		fmt.Println(err)
	}
	return outputFileName, err
}

func (s *ServersType) checkNow(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("выполняется checkNow")
	s.check()
}

//checkLoop URL periodic
func (s *ServersType) checkLoop() {
	for {
		s.check()
		fmt.Println("Ждём ")
		fmt.Println("================================================================================")
		time.Sleep(TimeOutSleep)
	}
}

func (s *ServersType) check() {
	for name, serverElm := range s.data {
		serverElm.checkElement()
		s.data[name] = serverElm
	}
}

func main() {
	intro()
	// Загружаю конфиг
	servers.LoadServers(serversFileName)
	go servers.checkLoop()
	http.HandleFunc("/", servers.myhandler)
	http.HandleFunc("/getreport", servers.getreport)
	http.HandleFunc("/checknow", servers.checkNow)
	http.Handle("/report/", http.StripPrefix("/report/", http.FileServer(http.Dir("./report"))))
	fmt.Println("Запуск локального WEB-сервера на порту :8088")
	fmt.Println("================================================================================")
	http.ListenAndServe(":8088", nil)
}
