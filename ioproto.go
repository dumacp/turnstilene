package turnstilene

import (
	"fmt"
	"sync"

	"github.com/tarm/serial"
)

//DeviceIO interface to deviceIO
type DeviceIO interface {
	SendFrame(funtion, bank byte, data []byte, len int) error
	ReadData(bank byte, len int) ([]byte, error)
	SetAddress(addres byte)
}

type deviceIO struct {
	address byte
	port    *serial.Port
	mux     sync.Mutex
}

//NewDeviceIO create new deviceIO
func NewDeviceIO(port *serial.Port) (DeviceIO, error) {
	dev := &deviceIO{}
	dev.port = port
	return dev, nil
}

func (d *deviceIO) SetAddress(addres byte) {
	d.address = addres
}

func csum(data []byte) byte {
	csum := byte(0)
	for _, v := range data {
		csum = (csum ^ v) & 0xFF
	}
	csum = (-csum & 0xFF)
	return csum
}

func (d *deviceIO) SendFrame(funtion, bank byte, data []byte, len int) error {
	//trama = [@address, funtion, bank, len]

	frame := []byte{d.address, funtion, bank, byte(len)}
	if data != nil {
		frame = append(frame, data...)
	}

	xsum := csum(frame)
	frame = append(frame, xsum)
	frame = append(frame, byte(0xFC))
	// fmt.Printf("sendframe: [% X]\n", frame)
	_, err := d.port.Write(frame)
	if err != nil {
		return err
	}
	return nil
}

func verify(data []byte) error {
	if data[(len(data)-1)] != 0xFC {
		return fmt.Errorf("bad response")
	}
	if len(data) < 6 {
		return fmt.Errorf("bad response")
	}
	xsum := csum(data[:len(data)-3])
	if xsum != data[len(data)-2] {
		return fmt.Errorf("bad response, checksum error")
	}
	return nil

}

func (d *deviceIO) ReadData(bank byte, length int) ([]byte, error) {
	d.mux.Lock()
	defer d.mux.Unlock()
	d.SendFrame(0x10, bank, nil, length)
	buf := make([]byte, 128)
	n, err := d.port.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, fmt.Errorf("Error read data, deviceIo")
	}
	if n == 0 {
		return nil, fmt.Errorf("not data, deviceIo")
	}
	if err := verify(buf[:n]); err != nil {
		return nil, err
	}

	resp := buf[:n]

	// fmt.Printf("sendframe resp: [% X]\n", resp)

	return resp[4 : len(resp)-2], nil
}
