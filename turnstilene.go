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
	Registers() ([]uint32, error)
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
	io.SetAddress(0x82)

	dev := &device{
		io,
	}

	return dev, nil
}

//Listen listen device
func (d *device) Listen() chan Event {

	first := true

	lenData := 14 //length data

	memArray := make([]byte, lenData)
	t1 := time.Tick(1 * time.Second)
	ch := make(chan Event, 4)

	var memInputs = struct {
		inputA  uint32
		inputB  uint32
		failure uint32
		battery bool
		alarm   bool
	}{}

	go func() {
		for {
			select {
			case <-t1:
				resp, err := d.io.ReadData(byte(0x10), lenData)
				// fmt.Printf("sendframe resp: [% X]\n", resp)
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

				alarm := false
				if resp[12] > 0x00 {
					alarm = true
				}
				battery := false
				if resp[13] > 0x00 {
					battery = true
				}
				if first {
					memInputs.inputA = inputA
					memInputs.inputB = inputB
					memInputs.failure = failure
					memInputs.battery = battery
					memInputs.alarm = alarm
					first = false
				}
				if inputA != memInputs.inputA {
					select {
					case ch <- Event{
						Type:  InputA,
						Value: inputA,
					}:
					default:
						log.Println("timeout send event")
					}
					memInputs.inputA = inputA
				}
				if inputB != memInputs.inputB {
					select {
					case ch <- Event{
						Type:  InputB,
						Value: inputB,
					}:
					default:
						log.Println("timeout send event")
					}
					memInputs.inputB = inputB
				}
				if failure != memInputs.failure {
					select {
					case ch <- Event{
						Type:  Failure,
						Value: failure,
					}:
					default:
						log.Println("timeout send event")
					}
					memInputs.failure = failure
				}
				if battery != memInputs.battery {
					select {
					case ch <- Event{
						Type:  Battery,
						Value: battery,
					}:
					default:
						log.Println("timeout send event")
					}
					memInputs.battery = battery
				}
				if alarm != memInputs.alarm {
					select {
					case ch <- Event{
						Type:  Alarm,
						Value: alarm,
					}:
					default:
						log.Println("timeout send event")
					}
					memInputs.alarm = alarm
				}
			}
		}
	}()
	return ch
}

func (d *device) Registers() ([]uint32, error) {
	lenData := 14 //length data
	resp, err := d.io.ReadData(byte(0x10), lenData)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	inputA := binary.LittleEndian.Uint32(resp[0:4])
	inputB := binary.LittleEndian.Uint32(resp[4:8])
	failure := binary.LittleEndian.Uint32(resp[8:12])
	alarm := false
	if resp[12] > 0x00 {
		alarm = true
	}
	battery := false
	if resp[13] > 0x00 {
		battery = true
	}
	vreg := []uint32{inputA, inputB, failure}
	if battery {
		vreg = append(vreg, 1)
	} else {
		vreg = append(vreg, 0)
	}
	if alarm {
		vreg = append(vreg, 1)
	} else {
		vreg = append(vreg, 0)
	}
	return vreg, nil
}
