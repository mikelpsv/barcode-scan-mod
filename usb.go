package usb

import (
	"fmt"
	"github.com/mikelpsv/gousb"
)

/*
// Tested device
//
// Device: Voyager-1200g
// Vendor: 0x0c2e Metrologic Instruments
// Product:
//   0x0a01: эмуляция клавиатуры
//   0x0a0a: COM
//   0x0a07: HID
// wMaxPacketSize     0x0040  1x 64 bytes
*/

const (
	ScannerModeUnknown = 0
	/*
		bInterfaceClass 		3 Human Interface Device
		bInterfaceSubClass 		1 Boot Interface Subclass
		bInterfaceProtocol      1 Keyboard
		iInterface             	15 HID Keyboard Emulation
	*/
	ScannerModeHIDKeyboardEmulation = 1
	/*
		bDeviceClass            0
		bInterfaceClass         3 Human Interface Device
		bInterfaceSubClass      0
		bInterfaceProtocol      0
		iInterface             33 HID POS
	*/
	ScannerModeHIDDevice = 2
	/*
	   bDeviceClass          239 Miscellaneous Device
	   bInterfaceClass         2 Communications
	   bInterfaceSubClass      2 Abstract (modem)
	   bInterfaceProtocol      1 AT-commands
	   iInterface             41 CDC-ACM Comm
	*/
	ScannerModeCOMEmulation = 3
)

type ScannerMode uint8

func (sm ScannerMode) String() string {
	return scannerModeDescription[sm]
}

var scannerModeDescription = map[ScannerMode]string{
	ScannerModeUnknown:              "unknown",
	ScannerModeHIDKeyboardEmulation: "keyboard",
	ScannerModeHIDDevice:            "hid",
	ScannerModeCOMEmulation:         "com",
}

// Scanner is a representation of a barcode scanner.
type Scanner struct {
	// open device
	*gousb.Device
	// device info
	Info ScannerInfo
}

// ScannerInfo this is extended information of Scanner
type ScannerInfo struct {
	Config        int
	Interface     int
	Setup         int
	Endpoint      int
	Class         gousb.Class
	SubClass      gousb.Class
	Protocol      gousb.Protocol
	MaxPacketSize int
	Mode          ScannerMode
}

// determineMode defines the scanner operation mode
func (di *ScannerInfo) determineMode() {
	di.Mode = ScannerModeUnknown
	if di.Class == 3 && di.SubClass == 1 && di.Protocol == 1 {
		di.Mode = ScannerModeHIDKeyboardEmulation
	}
	if di.Class == 3 && di.SubClass == 0 && di.Protocol == 0 {
		di.Mode = ScannerModeHIDDevice
	}
	if di.Class == 2 && di.SubClass == 2 && di.Protocol == 1 {
		di.Mode = ScannerModeCOMEmulation
	}
}

// DeviceDesc is an extended representation of a USB device descriptor (gousb.DeviceDesc).
type DeviceDesc struct {
	ManufacturerDesc string
	ProductDesc      string
	Serial           string
	gousb.DeviceDesc
}

// GetScanner returns a device with vendor, product and serial values
func (dd *DeviceDesc) GetScanner(ctx *gousb.Context) (*Scanner, error) {
	return GetScanner(ctx, dd.Vendor, dd.Product, dd.Serial)
}

func NewUsbContext() *gousb.Context {
	return gousb.NewContext()
}

// GetUsbDevices enum all usb devices
func GetUsbDevices(ctx *gousb.Context) ([]DeviceDesc, error) {
	devList := make([]DeviceDesc, 0)
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		// TODO: It may be necessary to set a device here that is not exactly an input device. Filter by class and subclass.
		return true
	})
	if err != nil {
		for _, d := range devs {
			d.Close()
		}
		return devList, err
	}
	for _, d := range devs {
		dd := DeviceDesc{}
		dd.ManufacturerDesc, _ = d.Manufacturer()
		dd.ProductDesc, _ = d.Product()
		dd.Serial, _ = d.SerialNumber()
		dd.DeviceDesc = *d.Desc
		devList = append(devList, dd)
		d.Close()
	}
	return devList, nil
}

// GetScanner returns a device with vendor, product and serial values
func GetScanner(ctx *gousb.Context, Vendor gousb.ID, Product gousb.ID, Serial string) (*Scanner, error) {
	var dev *gousb.Device
	var err error

	if Serial == "" {
		dev, err = ctx.OpenDeviceWithVIDPID(Vendor, Product)
	} else {
		dev, err = ctx.OpenDeviceWithVIDPIDSerial(Vendor, Product, Serial)
	}

	if err != nil {
		return nil, err
	}

	dev.SetAutoDetach(true)

	retScan := Scanner{}
	for _, cfg := range dev.Desc.Configs {
		for _, alt := range cfg.Interfaces {
			for _, iface := range alt.AltSettings {
				for _, end := range iface.Endpoints {
					if end.Direction == gousb.EndpointDirectionIn {
						retScan.Info.Config = cfg.Number
						retScan.Info.Interface = alt.Number
						retScan.Info.Setup = iface.Number
						retScan.Info.Endpoint = end.Number
						retScan.Info.Class = iface.Class
						retScan.Info.SubClass = iface.SubClass
						retScan.Info.Protocol = iface.Protocol
						retScan.Device = dev
						retScan.Info.MaxPacketSize = end.MaxPacketSize
						retScan.Info.determineMode()
						return &retScan, nil
					}
				}
			}
		}
	}
	return &retScan, fmt.Errorf("")
}

func (s *Scanner) Read() ([]byte, error) {
	cfg, err := s.Device.Config(s.Info.Config)
	intf, err := cfg.Interface(s.Info.Interface, s.Info.Setup)
	ep, err := intf.InEndpoint(s.Info.Endpoint)
	if err != nil {
		return nil, err
	}

	readBytes := 0
	buf := make([]byte, ep.Desc.MaxPacketSize)
	if s.Info.Mode == ScannerModeHIDDevice {
		readBytes, err = ep.Read(buf)
		if err != nil {
			return nil, err
		}
		if readBytes == 0 {
			return nil, fmt.Errorf("read returned 0 bytes of data")
		}
	} else if s.Info.Mode == ScannerModeHIDKeyboardEmulation {
		// TODO: change buffer, read while not found data terminator and convert fata to []byte
		for {
			readBytes, err = ep.Read(buf)
			if err != nil {
				return nil, err
			}
			if readBytes == 0 {
				return nil, fmt.Errorf("read returned 0 bytes of data")
				break
			}
		}
	}

	return buf[:readBytes], nil
}
