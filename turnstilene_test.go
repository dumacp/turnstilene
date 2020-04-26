package turnstilene

import (
	"reflect"
	"testing"
)

func TestTurnstile(t *testing.T) {
	type args struct {
		port     string
		baudRate int
	}
	tests := []struct {
		name    string
		args    args
		want    Device
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Turnstile(tt.args.port, tt.args.baudRate)
			if (err != nil) != tt.wantErr {
				t.Errorf("Turnstile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Turnstile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_device_Listen(t *testing.T) {
	type fields struct {
		io DeviceIO
	}
	tests := []struct {
		name   string
		fields fields
		want   chan Event
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &device{
				io: tt.fields.io,
			}
			if got := d.Listen(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("device.Listen() = %v, want %v", got, tt.want)
			}
		})
	}
}
