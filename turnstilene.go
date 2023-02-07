package turnstilene

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"time"

	"github.com/tarm/serial"
)

// Events enum
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
	//Error Event
	Error
)

const (
	LimitErrors = 3
)

// Event event data
type Event struct {
	Type  Events
	Value interface{}
}

// Device interface to device
type Device interface {
	Listen() chan Event
	Registers() ([]uint32, error)
	ReadTimeout() time.Duration
}

type device struct {
	io     DeviceIO
	config *serial.Config
}

// Turnstile create new device
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
	if err != nil {
		return nil, err
	}
	io.SetAddress(0x82)

	dev := &device{
		io:     io,
		config: config,
	}

	return dev, nil
}

// Listen listen device
func (d *device) Listen() chan Event {

	first := true

	lenData := 14 //length data

	memArray := make([]byte, lenData)
	t1 := time.NewTicker(1 * time.Second)
	defer t1.Stop()
	ch := make(chan Event, 4)

	var memInputs = struct {
		inputA  uint32
		inputB  uint32
		failure uint32
		battery bool
		alarm   bool
	}{}

	memCountErrors := 0

	go func() {
		defer close(ch)
		for range t1.C {
			tr := time.Now()
			resp, err := d.io.ReadData(byte(0x10), lenData)
			// fmt.Printf("sendframe resp: [% X]\n", resp)

			if err != nil {
				log.Println(err)
				if memCountErrors > LimitErrors {
					select {
					case ch <- Event{
						Type:  Error,
						Value: err,
					}:
					case <-time.After(3 * time.Second):
						log.Println("timeout send event")
					}
					return
				}
				if errors.Is(err, io.EOF) {
					if len(resp) <= 0 && time.Since(tr) < d.ReadTimeout()/20 {
						memCountErrors++
						continue

					}
				} else {
					memCountErrors++
					continue
				}
			}

			eq := true
			for i, v := range resp {
				if v != memArray[i] {
					eq = false
					break
				}
			}
			for i, v := range resp {
				memArray[i] = v
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
			if inputA > memInputs.inputA && inputA-memInputs.inputA < 30 {
				select {
				case ch <- Event{
					Type:  InputA,
					Value: inputA,
				}:
				case <-time.After(3 * time.Second):
					log.Println("timeout send event")
				}

			} else {
				if !first {
					log.Printf("inputA-memInputs.inputA is greater than 30: %d - %d",
						inputA, memInputs.inputA)
				}
			}
			memInputs.inputA = inputA
			if inputB > memInputs.inputB && inputB-memInputs.inputB < 30 {
				select {
				case ch <- Event{
					Type:  InputB,
					Value: inputB,
				}:
				case <-time.After(3 * time.Second):
					log.Println("timeout send event")
				}

			} else {
				if !first {
					log.Printf("inputB-memInputs.inputB is greater than 30: %d - %d",
						inputB, memInputs.inputB)
				}
			}
			memInputs.inputB = inputB

			if failure != memInputs.failure {
				select {
				case ch <- Event{
					Type:  Failure,
					Value: failure,
				}:
				case <-time.After(3 * time.Second):
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
				case <-time.After(3 * time.Second):
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
				case <-time.After(3 * time.Second):
					log.Println("timeout send event")
				}
				memInputs.alarm = alarm
			}
		}
	}()
	return ch
}

func (d *device) ReadTimeout() time.Duration {
	if d.config != nil {
		return d.config.ReadTimeout
	}
	return 0
}

func (d *device) Registers() ([]uint32, error) {
	lenData := 14 //length data
	t1 := time.Now()
	resp, err := d.io.ReadData(byte(0x10), lenData)

	if err != nil {
		log.Println(err)
		if errors.Is(err, io.EOF) {
			if len(resp) <= 0 && time.Since(t1) < d.ReadTimeout()/20 {
				return nil, err
			}
		} else {
			return nil, err
		}
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
