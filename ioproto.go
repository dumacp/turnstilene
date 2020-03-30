package turnstilene

import (
	"fmt"

	"github.com/tarm/serial"
)

//DeviceIO interface to deviceIO
type DeviceIO interface {
	SendFrame(funtion, bank byte, data []byte) error
	ReadData(bank byte, len int) ([]byte, error)
}

type deviceIO struct {
	address byte
	port    *serial.Port
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

func (d *deviceIO) SendFrame(funtion, bank byte, data []byte) error {
	//trama = [@address, funtion, bank, len]
	frame := []byte{d.address, funtion, bank}
	if data != nil {
		frame = append(frame, data...)
	}

	xsum := csum(frame)
	frame = append(frame, xsum)
	frame = append(frame, byte(0xFC))
	_, err := d.port.Write(frame)
	if err != nil {
		return err
	}
	return nil
}

func (d *deviceIO) ReadData(bank byte, len int) ([]byte, error) {
	d.SendFrame(0x10, bank, nil)
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
	return buf[:n], nil
}
