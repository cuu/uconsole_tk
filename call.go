/*
 * ./call --number=1234567
 * 
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"reflect"
	//"sync"
	"time"
	"github.com/maltegrosse/go-modemmanager"
	"github.com/visualfc/atk/tk"
	"strings"
)

type Window struct {
	*tk.Window
	StatLabel *tk.Label
	
}

var number string
var current_call = make(chan modemmanager.Call, 1)
var global_modem modemmanager.Modem

func pipeNewCall( call modemmanager.Call ) {
	select {
	case current_call <- call: // set current call
		fmt.Println("Set current call to channel")
	default:
		fmt.Println("set call channel failed!")
	}

}

func GetCallNumber( call modemmanager.Call) string {
	str, err := call.GetNumber()
	if err != nil {
		return ""
	}
	return str
}

func listenToModemVoiceCallAdded(modem modemmanager.Modem, window *Window) {
	// listen new calls
	voice, err := modem.GetVoice()
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(voice.GetObjectPath())
	c := voice.SubscribeCallAdded()
	fmt.Println("start listening ....")

	for v := range c {
		fmt.Println("CallAdded ", v)
		fmt.Println(reflect.TypeOf(v))
		fmt.Println("name", v.Name)
		fmt.Println("path", v.Path)
		fmt.Println("body", v.Body)
		fmt.Println("listenToModemVoiceCallAdded sender", v.Sender)

		if strings.Contains(v.Name, modemmanager.ModemVoiceSignalCallAdded) == true {

			calls, err := voice.ParseCallAdded(v)
			if err == nil {
				fmt.Println("newCall()")
				
				state, err := calls.GetState()
				if err == nil {
					if state == modemmanager.MmCallStateUnknown || state == modemmanager.MmCallStateRingingOut {
						pipeNewCall(calls)
						//change label
						tk.Async(func() {
							window.StatLabel.SetText("Ringing....")
						})

						state_changed := calls.SubscribeStateChanged()
						fmt.Println("newCallOut() wait call state change")
						for val := range state_changed {
							fmt.Println(" call.SubscribeStateChanged ",val)
							fmt.Println(reflect.TypeOf(val))
							fmt.Println("name", val.Name)
							fmt.Println("path", val.Path)
							fmt.Println("body", val.Body)
							fmt.Println("sender", val.Sender)
							oldState, newState, reason, err := calls.ParseStateChanged(val)
							if err == nil {
								fmt.Println("oldState:", oldState)
								fmt.Println("newState:",newState)
								fmt.Println("reason:",reason)

								if newState == modemmanager.MmCallStateActive {
									tk.Async(func() {
										window.StatLabel.SetText("Talking....")
									})
								}
								if newState ==  modemmanager.MmCallStateWaiting {
									tk.Async(func() {
										window.StatLabel.SetText("Waiting to pick...")
									})
								}
								
								if newState == modemmanager.MmCallStateTerminated {
									tk.Async(func() {
										window.StatLabel.SetText("Call rejected...exiting...")
									})
									log.Fatal(fmt.Sprintf("Call %s end", GetCallNumber(calls)))
									break
								}
							}
						}
		
					}
				} else {
					fmt.Println(err)
				}
			} else {
				fmt.Println(err)
			}
		}

	}
}

func InitModem(window *Window) modemmanager.Modem {

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
		go listenToModemVoiceCallAdded(modem, window)
		global_modem = modem
		break
	}
	maxVolume()
	return global_modem

}

func createCall(window*Window,number string) {

	time.Sleep(time.Second * time.Duration(3))
	
	InitModem(window)
	
	voice, err := global_modem.GetVoice()
	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println(voice.GetObjectPath())

	if call, err := voice.CreateCall(number); err == nil {
		call.Start()
	} else {
		log.Fatal("create call failed", err)
	}
}

func NewWindow() *Window {
	mw := &Window{}
	mw.Window = tk.RootWindow()
	lbl := tk.NewLabel(mw, fmt.Sprintf("Calling ...%s", number))
	lbl_stats := tk.NewLabel(mw, "Calling")
	
	btn := tk.NewButton(mw, "Quit")
	btn.OnCommand(func() {
		rejectCall()
		tk.Quit()
	})

	mw.StatLabel = lbl_stats
	
	tk.NewVPackLayout(mw).AddWidgets(lbl,mw.StatLabel, tk.NewLayoutSpacer(mw, 0, true), btn)
	mw.ResizeN(300, 100)
	return mw
}

func rejectCall() error {

	select {
	case call := <-current_call:
		//fmt.Println("get call from channel")
		err := call.Hangup() //requires sudo
		if err != nil {
			fmt.Println(err)
		}
	default:
		fmt.Println("no call object")
	}

	return nil
}

func maxVolume() error {
	atcmd := "AT+CLVL=5"
	_,err := global_modem.Command(atcmd,1)
	return err
}

func main() {

	word := flag.String("number", "", "a phone number")
	flag.Parse()
	if len(*word) <= 0 {
		log.Fatal("no phone number, --number=123456 to specific a valid phone number")
	}
	number = *word
	
	tk.Init()
	mw := NewWindow()
	mw.SetTitle("uConsole phone call")
	mw.Center(nil)
	mw.ShowNormal()
	
	tk.MainLoop(func() {
		createCall(mw,number)
		
	})
}
