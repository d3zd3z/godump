// Progress meter and logging.

// The progress meter allows a periodically updating meter to be shown
// that will indicate progress through a backup operation.  The meter
// grabs logging input as well so that the meter and log output will
// always be coordinated.  Output directly to stdout will not be
// coordinated, and this should be managed carefully.
package meter

import (
	"fmt"
	"log"
	"os"
	"time"
)

// An Informer is something that is able to provide a status display.
// The display consists of one or more lines of text (the strings
// should not contain embedded newlines).
type Informer interface {
	// Retrieve the current info status for the progress meter.
	GetMeter() []string
}

// Initialize the progress meter.  Spawns off a thread to show the
// output, and captures the log output.
func Setup() {
	main.log = make(chan string)
	main.meter = make(chan []string)
	main.done = make(chan bool)
	log.SetOutput(&main)
	main.tick = time.Tick(time.Second)

	go main.Run()
}

// Stop the logging.  Displays any pending messages before returning.
// After this, logging just goes to stdout as normal.
func Shutdown() {
	close(main.log)
	log.SetOutput(os.Stdout)
	<-main.done
}

// Indicate that the information in the log display might be updated.
// If force is false, the display will only be updated if a certain
// amount of time has elapsed, otherwise, it will be updated
// immediately.
func Sync(inform Informer, force bool) {
	select {
	case <-main.tick:
		force = true
	default:
	}

	if force && main.meter != nil {
		main.meter <- inform.GetMeter()
	}
}

type meter struct {
	log   chan string
	tick  <-chan time.Time
	meter chan []string

	// Make sure the last message gets out.
	done chan bool

	// Last shown message.
	msg []string
}

func (self *meter) Run() {
	for {
		select {
		case msg, ok := <-self.log:
			if ok {
				// Show message.
				self.Clear()
				fmt.Print(msg)
				self.Show()
			} else {
				// The log is closed.
				self.done <- true
				return
			}
		case msg := <-self.meter:
			self.Clear()
			self.msg = msg
			self.Show()
		}
	}
}

// The main meter.
var main meter

func (self *meter) Write(p []byte) (n int, err error) {
	// The messages are always single lines, with the trailing
	// newline.
	self.log <- string(p)
	n = len(p)
	return
}

func (self *meter) Clear() {
	if self.msg == nil {
		return
	}

	fmt.Printf("\x1b[%dF\x1b[J", len(self.msg))
}

func (self *meter) Show() {
	if self.msg == nil {
		return
	}
	for _, line := range self.msg {
		fmt.Println(line)
	}
}
