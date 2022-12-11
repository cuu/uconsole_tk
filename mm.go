package main

import (
	"fmt"
	"log"
	"github.com/maltegrosse/go-modemmanager"
)

var global_modem modemmanager.Modem

func InitModem(){
	mmgr, err := modemmanager.NewModemManager()
	if err != nil {
		log.Fatal(err.Error())
	}
	version, err := mmgr.GetVersion()
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println("ModemManager Version: ", version)
	modems, err := mmgr.GetModems()
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, modem := range modems {
		fmt.Println("ObjectPath: ", modem.GetObjectPath())
		global_modem = modem
	}
}


