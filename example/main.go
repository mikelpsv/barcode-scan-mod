package main

import (
	"fmt"
	"github.com/mikelpsv/barcode-scan-mod"
)

func main() {
	ctx := usb.NewUsbContext()
	defer ctx.Close()

	devList, err := usb.GetUsbDevices(ctx)
	if err != nil {
		fmt.Printf("Get devices failed, err %v", err)
		return
	}
	fmt.Printf("Found %d USB devices", len(devList))

	selDev := usb.DeviceDesc{}
	for _, dev := range devList {
		fmt.Printf("Bus: %d, Addr: %d, Port: %d, Speed: %d, Vendor: %x, Product: %s, Class %s, SubClass %s, Protocol %d\n",
			dev.Bus, dev.Address, dev.Port, dev.Speed, dev.Vendor, dev.Product, dev.Class, dev.SubClass, dev.Protocol)
		if dev.Vendor == 0x0c2e {
			selDev = dev
		}
	}

	scanner, err := selDev.GetScanner(ctx)
	defer scanner.Close()

	if err != nil {
		fmt.Println(err)
	}

	data, err := scanner.Read()
	if err != nil {
		fmt.Printf("Data read failed, err %v", err)
		return
	}
	fmt.Println(string(data))

}
