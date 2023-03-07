package main

import (
	"context"
	"fmt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"inputs/sni"
	"log"
	"time"
)

func main() {
	log.SetFlags(log.Lmicroseconds | log.LUTC)

	var err error

	var conn *grpc.ClientConn
	conn, err = grpc.Dial("localhost:8191", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := sni.NewDevicesClient(conn)
	var rsp *sni.DevicesResponse
	rsp, err = client.ListDevices(context.Background(), &sni.DevicesRequest{})
	if err != nil {
		log.Fatalf("fail to list devices: %v", err)
	}

	for i, device := range rsp.Devices {
		fmt.Printf("[%d]: %+v\n", i, device)
	}

	memory := sni.NewDeviceMemoryClient(conn)
	read, err := memory.StreamRead(context.Background())
	if err != nil {
		return
	}

	mr := &sni.MultiReadMemoryRequest{
		Uri: rsp.Devices[0].Uri,
		Requests: []*sni.ReadMemoryRequest{
			{
				RequestAddress:       0xf90718,
				RequestAddressSpace:  sni.AddressSpace_FxPakPro,
				RequestMemoryMapping: sni.MemoryMapping_LoROM,
				Size:                 2,
			},
		},
	}

	ep := message.NewPrinter(language.English)

	for {
		start := time.Now()
		err = read.Send(mr)
		if err != nil {
			log.Fatal(err)
		}
		var mrsp *sni.MultiReadMemoryResponse
		mrsp, err = read.Recv()
		end := time.Now()
		if err != nil {
			log.Fatal(err)
		}
		delay := end.Sub(start).Microseconds()

		b0 := mrsp.Responses[0].Data[0]
		b1 := mrsp.Responses[0].Data[1]
		//fmt.Printf("%s %08b %08b\n", ep.Sprintf("%6d", delay), b0, b1)

		s0 := []byte("axlr----")
		s1 := []byte("bycsudlr")
		for i, k := 0, byte(128); i < 4; i, k = i+1, k>>1 {
			if b0&k != 0 {
				s0[i] &= ^byte(0x20)
			}
		}
		for i, k := 0, byte(128); i < 8; i, k = i+1, k>>1 {
			if b1&k != 0 {
				s1[i] &= ^byte(0x20)
			}
		}
		fmt.Printf(
			"%s    %s %s\n",
			ep.Sprintf("%6d", delay),
			s0,
			s1,
		)
	}

	err = read.CloseSend()
	if err != nil {
		log.Fatal(err)
	}
}
