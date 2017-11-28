package main

import (
	"container/list"
	"errors"
	"log"
	"net"
	"sync"
)

/*GatewayClient coap client connection description. Only for server side. */
type GatewayClient struct {
	con *net.UDPConn

	realAddr       *net.UDPAddr
	connectionAddr *net.UDPAddr
}

var (
	clientList  = list.New()
	clientMutex = sync.Mutex{}
)

func getClientByConAddr(addr *net.UDPAddr) *GatewayClient {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	for e := clientList.Front(); e != nil; e = e.Next() {
		if tn, ok := e.Value.(GatewayClient); ok {
			if tn.connectionAddr.IP.Equal(addr.IP) &&
				tn.connectionAddr.Port == addr.Port {

				return &tn
			}
		}
	}

	return nil
}
func getClientByRealAddr(addr *net.UDPAddr) *GatewayClient {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	for e := clientList.Front(); e != nil; e = e.Next() {
		if tn, ok := e.Value.(GatewayClient); ok {
			if tn.realAddr.IP.Equal(addr.IP) &&
				tn.realAddr.Port == addr.Port {

				return &tn
			}
		}
	}

	return nil
}

func appendClient(gatewayClient *GatewayClient) error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	for e := clientList.Front(); e != nil; e = e.Next() {
		if tn, ok := e.Value.(GatewayClient); ok {
			if tn == *gatewayClient {
				return errors.New("already exists")
			}
		}
	}

	clientList.PushBack(*gatewayClient)

	return nil
}

func removeClient(gatewayClient *GatewayClient) error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	for e := clientList.Front(); e != nil; e = e.Next() {
		if tn, ok := e.Value.(GatewayClient); ok {
			if tn == *gatewayClient {
				clientList.Remove(e)
				return nil
			}
		}
	}

	return errors.New("gatewayClient not found")
}

func printClients() {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	counter := 0
	for e := clientList.Front(); e != nil; e = e.Next() {
		if tn, ok := e.Value.(GatewayClient); ok {
			log.Printf("[%d]: connectionAddr: %v ; realAddr: %v\n", counter, tn.connectionAddr, tn.realAddr)
			counter++
		}
	}
}

func removeClientByConAddr(addr *net.UDPAddr) error {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	for e := clientList.Front(); e != nil; e = e.Next() {
		if tn, ok := e.Value.(GatewayClient); ok {
			if tn.connectionAddr.IP.Equal(addr.IP) &&
				tn.connectionAddr.Port == addr.Port {

				clientList.Remove(e)
				return nil
			}
		}
	}

	return errors.New("gatewayClient not found")
}
