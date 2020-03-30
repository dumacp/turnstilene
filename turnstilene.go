package turnstilene

import (
	"encoding/binary"
	"log"
	"time"

	"github.com/tarm/serial"
)

//Events enum
type Events int

const (
	//InputA Event
	InputA Events = iota
	//InputB Event
	InputB
	//Failure Event
	Failure
	//Alarm Event
	Alarm
	//Battery Event
	Battery
)

//Event event data
type Event struct {
	Type  Events
	Value interface{}
}

//Device interface to device
type Device interface {
	Listen() chan Event
}

type device struct {
	io DeviceIO
}

//Turnstile create new device
func Turnstile(port string, baudRate int) (Device, error) {
	config := &serial.Config{
		Name:        port,
		Baud:        baudRate,
		ReadTimeout: 600 * time.Second,
	}

	p, err := serial.OpenPort(config)
	if err != nil {
		return nil, err
	}

	io, err := NewDeviceIO(p)

	dev := &device{
		io,
	}

	return dev, nil
}

//Listen listen device
func (d *device) Listen() chan Event {

	memInputs := struct {
		inputA  uint32
		inputB  uint32
		failure uint32
		battery bool
		alarm   bool
	}{}

	lenData := 14 //length data

	memArray := make([]byte, lenData)
	t1 := time.Tick(1 * time.Second)
	ch := make(chan Event, 0)

	go func() {
		for {
			select {
			case <-t1:
				resp, err := d.io.ReadData(byte(0x10), lenData)
				if err != nil {
					log.Println(err)
					continue
				}

				eq := true
				for i, v := range resp {
					if v != memArray[i] {
						eq = false
						break
					}
				}
				if eq {
					continue
				}
				if len(resp) < lenData {
					continue
				}
				inputA := binary.LittleEndian.Uint32(resp[0:4])
				inputB := binary.LittleEndian.Uint32(resp[4:8])
				failure := binary.LittleEndian.Uint32(resp[8:12])
				battery := false
				if resp[12] > 0x00 {
					battery = true
				}
				alarm := false
				if resp[12] > 0x00 {
					alarm = true
				}
				if inputA != memInputs.inputA {
					select {
					case ch <- Event{
						Type:  InputA,
						Value: inputA,
					}:
					default:
						log.Println("timeot send event")
					}
				}
				if inputB != memInputs.inputB {
					select {
					case ch <- Event{
						Type:  InputB,
						Value: inputB,
					}:
					default:
						log.Println("timeot send event")
					}
				}
				if failure != memInputs.failure {
					select {
					case ch <- Event{
						Type:  Failure,
						Value: failure,
					}:
					default:
						log.Println("timeot send event")
					}
				}
				if battery != memInputs.battery {
					select {
					case ch <- Event{
						Type:  Battery,
						Value: battery,
					}:
					default:
						log.Println("timeot send event")
					}
				}
				if alarm != memInputs.alarm {
					select {
					case ch <- Event{
						Type:  Alarm,
						Value: alarm,
					}:
					default:
						log.Println("timeot send event")
					}
				}
			}
		}
	}()
	return nil
}
