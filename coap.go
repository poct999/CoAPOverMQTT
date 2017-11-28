package main

import (
	"fmt"
	"log"
	"net"
	"time"

	coap "github.com/dustin/go-coap"
)

var (
	clientListenerConnection *net.UDPConn
	serverAddr               *net.UDPAddr
)

func checkCoAPPayload(payload []byte) error {
	msg := coap.Message{}
	if err := msg.UnmarshalBinary(payload); err != nil {
		return err
	}

	return nil
}

func onCOAPClientMessage(l *net.UDPConn, a *net.UDPAddr, payload []byte) {
	log.Printf("new CoAP client message from: %v \n", a)

	if err := checkCoAPPayload(payload); err != nil {
		log.Printf("onCOAPClientMessage: Error on parsing udp packet: %v\n", err)
		return
	}

	topicName := getMQTTTopicNameFrom(a)
	publishMQTTMessage(topicName, payload)
	log.Printf("New CoAP client message was published to: %s\n", topicName)
}

func onCOAPServerMessage(l *net.UDPConn, a *net.UDPAddr, payload []byte) {
	log.Printf("new CoAP server message from: %v \n", l.LocalAddr().(*net.UDPAddr))

	if err := checkCoAPPayload(payload); err != nil {
		log.Printf("onCOAPClientMessage: Error on parsing udp packet: %v\n", err)
		return
	}

	gatewayClient := getClientByConAddr(l.LocalAddr().(*net.UDPAddr))
	if gatewayClient == nil {
		log.Printf("Error: gatewayClient not found...\n")
		return
	}

	topicName := getMQTTTopicNameTo(gatewayClient.realAddr)
	publishMQTTMessage(topicName, payload)
	log.Printf("New CoAP server message was published to: %s\n", topicName)
}

func listenConnection(con *net.UDPConn, callback func(l *net.UDPConn, conAddr *net.UDPAddr, payload []byte)) {
	buf := make([]byte, 1500)
	for {
		nr, addr, err := con.ReadFromUDP(buf)
		if err != nil {
			if neterr, ok := err.(net.Error); ok && (neterr.Temporary() || neterr.Timeout()) {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			return
		}
		tmp := make([]byte, nr)
		copy(tmp, buf)
		go callback(con, addr, tmp)
	}
}

func listenServerConnection(con *net.UDPConn, callback func(l *net.UDPConn, conAddr *net.UDPAddr, payload []byte)) {

	listenConnection(con, callback)

	conAddr := con.LocalAddr().(*net.UDPAddr)
	removeClientByConAddr(conAddr)
	log.Printf("Connection by addr '%v' was closed.", conAddr)
	con.Close()
}

func initNewCoApConnectionToServer(realClientAddr *net.UDPAddr) (*GatewayClient, error) {
	s, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return nil, err
	}

	gatewayClient := &GatewayClient{
		con:            s,
		realAddr:       realClientAddr,
		connectionAddr: s.LocalAddr().(*net.UDPAddr),
	}

	if err := appendClient(gatewayClient); err != nil {
		log.Printf("appendClient error: %v \n", err)
		return nil, err
	}

	log.Printf("newCoApConnectionToServer was init. addr: %v\n", gatewayClient.connectionAddr)
	log.Printf("realClientAddr: %v\n", gatewayClient.realAddr)

	go listenServerConnection(s, onCOAPServerMessage)

	return gatewayClient, nil
}

func listenCoAPClients() error {
	log.Printf("listenCoAPClients: start. [:%d]\n", listenPort)

	uaddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		return err
	}

	listener, err := net.ListenUDP("udp", uaddr)
	if err != nil {
		return err
	}

	clientListenerConnection = listener

	go listenConnection(listener, onCOAPClientMessage)

	return nil
}

func sendCOAPMessageToServer(con *net.UDPConn, payload []byte) error {
	writeLen, err := con.Write(payload)
	if err != nil {
		return err
	}
	if writeLen != len(payload) {
		return fmt.Errorf("Incorrect write value: %d, real value: %d", writeLen, len(payload))
	}

	log.Printf("message was sent to server: %v\n", serverAddr)

	return nil
}

func sendCOAPMessageToClient(addr *net.UDPAddr, payload []byte) error {
	writeLen, err := clientListenerConnection.WriteToUDP(payload, addr)
	if err != nil {
		return err
	}
	if writeLen != len(payload) {
		return fmt.Errorf("Incorrect write value: %d, real value: %d", writeLen, len(payload))
	}

	log.Printf("message was sent to client: %v\n", addr)

	return nil
}

func initCoAP() error {
	var err error
	if isServer {
		serverAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", coapIP, coapPort))
		if err != nil {
			return err
		}
	}

	if isClient {
		return listenCoAPClients()
	}

	return nil
}
