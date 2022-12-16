package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Option struct {
	Address    string
	TrialCount int
}

func main() {
	opt := options()
	ip, err := net.ResolveIPAddr("ip4", opt.Address)
	if err != nil {
		panic(err)
	}

	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	if opt.TrialCount == 0 {
		for {
			go try(c, ip.IP)
			time.Sleep(time.Second)
		}
	} else {
		for i := 0; i < opt.TrialCount; i++ {
			go try(c, ip.IP)
			time.Sleep(time.Second)
		}
	}
}

func options() Option {
	var option Option
	flag.StringVar(&option.Address, "a", "", "Destination address to which packets are sent.")
	flag.IntVar(&option.TrialCount, "n", 3, "Number of echo requests to send. If 0, it continues until user interrupt.")
	flag.Parse()
	if option.Address == "" {
		fmt.Println("Required address")
		os.Exit(1)
	}
	return option
}

func try(c *icmp.PacketConn, ip net.IP) {
	now := time.Now().UnixMilli()
	result := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(result, now)

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: result,
		},
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		panic(err)
	}
	if _, err := c.WriteTo(msgBytes, &net.IPAddr{IP: ip}); err != nil {
		panic(err)
	}

	c.SetDeadline(time.Now().Add(time.Second * 5))

	rb := make([]byte, 1500)
	n, _, err := c.ReadFrom(rb)
	if err != nil {
		fmt.Println("Receive Failed:", err.Error())
	} else {
		rm, err := icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), rb[:n])
		if err == nil && rm.Type == ipv4.ICMPTypeEchoReply {
			echo, ok := rm.Body.(*icmp.Echo)
			if !ok {
				fmt.Println("Body isn't echo:", err.Error())
			} else {
				t, _ := binary.Varint(echo.Data)
				fmt.Printf("%d ms\n", time.Now().UnixMilli()-t)
			}
		} else {
			fmt.Println("Parse Failed:", err.Error())
		}
	}
}
