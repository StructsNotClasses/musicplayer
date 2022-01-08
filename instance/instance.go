package instance

import (
	"github.com/StructsNotClasses/mim/instance/dirtree"
	"github.com/StructsNotClasses/mim/instance/notification"
	"github.com/StructsNotClasses/mim/instance/playback"
	"github.com/StructsNotClasses/mim/instance/client"
	"github.com/StructsNotClasses/mim/musicarray"
	"github.com/StructsNotClasses/mim/remote"

	gnc "github.com/rthornton128/goncurses"

	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"time"
)

const INT32MAX = 2147483647

type Script struct {
    name string
    contents []byte
}

type InputMode int

const (
	CommandMode InputMode = iota
	CharacterMode
)

type CommandStateMachine struct {
	line               []byte
	lines              []byte
	bindChar           gnc.Key
	onPlaybackBeingSet bool
    runOnNoPlayback Script
	mode               InputMode
	exit               bool
}

type CommandInterface struct {
    inWin *gnc.Window
    outWin *gnc.Window
    state           CommandStateMachine
    bindMap         map[gnc.Key]Script
    commandMap      map[string]string
}

type Instance struct {
    // windows
    backgroundWindow *gnc.Window

    // tree
	tree            dirtree.DirTree
    // command interface
    commandHandling    CommandInterface
    // mplayer management
	currentRemote   remote.Remote
	playbackState   playback.PlaybackState
	notifier        chan notification.Notification
    mpOutputWindow *gnc.Window
}

func New(scr *gnc.Window, array musicarray.MusicArray) Instance {
	if len(array) > INT32MAX {
		log.Fatal(fmt.Sprintf("Cannot play more than %d songs at a time.", INT32MAX))
	}
	rand.Seed(time.Now().UnixNano())

	//make user input non-blocking
	scr.Timeout(0)

	var instance Instance

    var treewin, inwin, outwin *gnc.Window
    var err error
	instance.backgroundWindow, instance.mpOutputWindow, treewin, inwin, outwin, err = client.New(scr)
	if err != nil {
		log.Fatal(err)
	}

	instance.tree = dirtree.New(treewin, array)

    instance.commandHandling = CommandInterface{
        inWin: inwin,
        outWin: outwin,
        state: CommandStateMachine{
            line: []byte{},
            lines: []byte{},
            bindChar:           0,
            onPlaybackBeingSet: false,
            mode:               CommandMode,
            exit:               false,
        },
        bindMap: make(map[gnc.Key]Script),
        commandMap: make(map[string]string),
    }

	instance.currentRemote = remote.Remote{}
	instance.notifier = make(chan notification.Notification)

	return instance
}

func (instance *Instance) LoadConfig(filename string) error {
	instance.commandHandling.state.line = []byte{}

	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	for _, b := range bytes {
		instance.HandleKey(gnc.Key(b))
	}
	return nil
}

func (instance *Instance) Exit() {
	instance.commandHandling.state.exit = true
	instance.StopPlayback()
}

func (i *Instance) StopPlayback() {
	if i.playbackState.PlaybackInProgress {
		i.currentRemote.SendString("quit\n")
		i.playbackState.ReceiveBlocking(i.notifier)
	}
}

func (c CommandInterface) InfoPrint(args ...interface{}) {
    c.outWin.Print(args...)
    c.outWin.Refresh()
}

func (c CommandInterface) InfoPrintln(args ...interface{}) {
    c.outWin.Println(args...)
    c.outWin.Refresh()
}

func (c CommandInterface) InfoPrintf(format string, args ...interface{}) {
    c.outWin.Printf(format, args...)
    c.outWin.Refresh()
}

func (c CommandInterface) InfoPrintRuntimeError() {
	if runtimeError := recover(); runtimeError != nil {
		c.InfoPrint(fmt.Sprintf("\nRuntime Error: %s\n", runtimeError))
	}
}

func (c CommandInterface) UpdateInput(currentLine []byte) {
	replaceCurrentLine(c.inWin, currentLine)
}

// replaceCurrentLine erases the current line on the window and prints a new one
// the new string's byte array could potentially contain a newline, which means this can replace the line with multiple lines
func replaceCurrentLine(win *gnc.Window, bs []byte) {
	s := string(bs)
	y, _ := win.CursorYX()
	_, w := win.MaxYX()
	win.HLine(y, 0, ' ', w)
	win.MovePrint(y, 0, s)
	win.Refresh()
}

func (i Instance) GetKey() gnc.Key {
	return i.backgroundWindow.GetChar()
}
