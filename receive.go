package main

import (
	//"flag"
	"fmt"
	"log"
	"reflect"
	//"sync"
	"os"
	"time"
	"github.com/maltegrosse/go-modemmanager"
	"github.com/visualfc/atk/tk"
	"strings"
	"strconv"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

type Window struct {
	*tk.Window
	StatLabel *tk.Label
	RejectBtn *tk.Button
	AcceptBtn *tk.Button
	MuteBtn   *tk.Button
}
var number string
var current_call = make(chan modemmanager.Call, 1)
var global_modem modemmanager.Modem

func play_ring(done chan bool) {
	fmt.Println("play_ring now")
	f, err := os.Open("telephone-ring-03a.mp3")
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	ctrl := &beep.Ctrl{Streamer: beep.Loop(-1, streamer), Paused: false}

	speaker.Play(ctrl)

	for {
		select {
		case <-done:
			return
		}
		/*
			speaker.Lock()
			ctrl.Paused = !ctrl.Paused
			speaker.Unlock()
		*/
	}
}

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

func listenToModemVoiceCallAdded(modem modemmanager.Modem, window*Window) {
	// listen new calls
	voice, err := modem.GetVoice()
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(voice.GetObjectPath())
	c := voice.SubscribeCallAdded()
	fmt.Println("start listening ....")

	for v := range c {
		fmt.Println("CallAdded ",v)
		fmt.Println(reflect.TypeOf(v))
		fmt.Println("name", v.Name)
		fmt.Println("path", v.Path)
		fmt.Println("body", v.Body)
		fmt.Println("listenToModemVoiceCallAdded sender", v.Sender)

		if strings.Contains(v.Name, modemmanager.ModemVoiceSignalCallAdded) == true {
		
			calls, err := voice.ParseCallAdded(v)
			if err == nil {
				fmt.Println("newCall()")
				state,err := calls.GetState()
				if err == nil{
					fmt.Println("newCall()",state)
					if state == modemmanager.MmCallStateRingingIn {
						pipeNewCall(calls)
						number = GetCallNumber(calls)
						tk.Async(func() {
							window.StatLabel.SetText(fmt.Sprintf("%s calling....",number))
						})
						ch_stop_ring := make(chan bool)
						go play_ring(ch_stop_ring)
						
						state_changed := calls.SubscribeStateChanged()
						fmt.Println("newCall() wait call state change")
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
								if newState == modemmanager.MmCallStateTerminated {
									ch_stop_ring <- true
									tk.Async(func() {
										window.StatLabel.SetText(fmt.Sprintf("%s Call rejected",number))
									})
									break
								}
							}
						}
						calls.Unsubscribe()
					}

				}else{
					fmt.Println(err)
				}
			}else {
				fmt.Println(err)
			}
		}
		
	}
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

func acceptCall() error {

	select {
	case call := <-current_call:
		//fmt.Println("get call from channel")
		err := call.Accept() //requires sudo
		if err != nil {
			fmt.Println(err)
		}
	default:
		fmt.Println("no call object")
	}

	return nil
}

func getMuteStatus() int {
	ret,err:= global_modem.Command("AT+CMUT?",1)
	fmt.Println(ret)
	if err == nil {
		arr := strings.Split(ret, "+CMUT:")
		mic := strings.TrimSpace(arr[1])
		mic = strings.Replace(mic, "'", "", -1)
		intVar,err := strconv.ParseInt(mic, 0, 10)
		if err != nil {
			fmt.Println(err)
		}
		
		return int(intVar)
	}else{
		fmt.Println(err)
	}
	return 0
}

func setMuteStatus(newStatus int) error {
	atcmd := fmt.Sprintf("AT+CMUT=%d",newStatus)
	fmt.Println(atcmd)
	ret,err := global_modem.Command(atcmd,1)
	fmt.Println(ret,err)
	return err
}

func muteMic(btn *tk.Button) {
	intVar := getMuteStatus()
	if intVar == 0 {
		err := setMuteStatus(1)
		if err == nil {
			btn.SetText("Unmute")
		}
	}else{
		err := setMuteStatus(0)
		if err == nil {
			btn.SetText("Mute")
		}
	}
}

func syncMute(window *Window) {
	intVar := getMuteStatus()
	labelStr := "Mute"
	if intVar == 1 {
		labelStr = "Unmute"
	}
	tk.Async(func() {
		window.MuteBtn.SetText(labelStr)
	})
		
}

func maxVolume() error {
	atcmd := "AT+CLVL=5"
	_,err := global_modem.Command(atcmd,1)
	return err
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
	//syncMute(window)
	maxVolume()
	return global_modem

}

func NewWindow() *Window {
	mw := &Window{}
	mw.Window = tk.RootWindow()
	vbox := tk.NewVPackLayout(mw)

	//lbl := tk.NewLabel(mw, fmt.Sprintf("%s Called", number))
	lbl_stats := tk.NewLabel(mw, "Waiting calls")
	mw.StatLabel = lbl_stats
		
        frm := tk.NewFrame(mw)
        frm.SetReliefStyle(tk.ReliefStyleFlat)
        frm.SetBorderWidth(5)

	mw.RejectBtn = tk.NewButton(frm, "Reject")
	mw.RejectBtn.OnCommand(func() {
		rejectCall()
		//tk.Quit()
	})
	mw.AcceptBtn = tk.NewButton(frm,"Accept")
	mw.AcceptBtn.OnCommand(func() {
		acceptCall()
	})
	
	hbox1 := tk.NewHPackLayout(frm)
	
	hbox1.AddWidgets(mw.AcceptBtn, mw.RejectBtn)
	hbox1.SetPaddingN(5,5)
	
	vbox.AddWidget(mw.StatLabel)
	vbox.AddWidget(tk.NewLayoutSpacer(mw, 0, true))
	vbox.AddWidget(frm)

	/*
	//Mute can not work
        frm2 := tk.NewFrame(mw)
        frm2.SetReliefStyle(tk.ReliefStyleFlat)
        frm2.SetBorderWidth(5)

	mw.MuteBtn = tk.NewButton(frm2, "Mute Mic")
	mw.MuteBtn.OnCommand(func() {
		muteMic(mw.MuteBtn)
	})
	
	hbox2 := tk.NewHPackLayout(frm2)
	
	hbox2.AddWidgets(mw.MuteBtn)
	hbox2.SetPaddingN(5,5)
	
	vbox.AddWidget(frm2)
	*/
        vbox.SetBorderWidth(10)
	vbox.Repack()
	
	mw.ResizeN(300, 150)
	return mw
}


func main() {
	
	tk.Init()
	mw := NewWindow()
	mw.SetTitle("uConsole phone receiver")
	mw.Center(nil)
	mw.ShowNormal()
	mw.OnClose(func() bool {
		fmt.Println("Closing window")
		return true
	})
	tk.MainLoop(func() {
		InitModem(mw)
	})
}
