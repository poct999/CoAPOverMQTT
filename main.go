package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	/* Arguments */
	mqttPort   int
	mqttIP     string
	coapPort   int
	coapIP     string
	listenPort int
	isClient   bool
	isServer   bool
)

/*
 * TODO list:
 * 1) keep alive
 * 2) mqtt configuration for keep alive
 */

func parseArgs() error {
	flag.StringVar(&mqttIP, "mqttIP", "127.0.0.1", "mqtt server ip")
	flag.IntVar(&mqttPort, "mqttPort", 1883, "mqtt server port")
	flag.StringVar(&coapIP, "coapIP", "127.0.0.1", "coap server ip")
	flag.IntVar(&coapPort, "coapPort", 5683, "coap server port")
	flag.IntVar(&listenPort, "listenPort", 4444, "server gateway port")
	flag.BoolVar(&isClient, "c", false, "is gateway in client side")
	flag.BoolVar(&isServer, "s", false, "is gateway in server side")

	flag.Parse()

	//Check args
	if !isClient && !isServer {
		return errors.New("Please, specify gateway side: -c for client side and -s for server side")
	}

	//Debug print args
	log.Printf("mqttIP: %v\n", mqttIP)
	log.Printf("mqttPort: %v\n", mqttPort)
	log.Printf("coapIP: %v\n", coapIP)
	log.Printf("coapPort: %v\n", coapPort)
	log.Printf("listenPort: %v\n", listenPort)
	log.Printf("isClient: %v\n", isClient)
	log.Printf("isServer: %v\n", isServer)

	return nil
}

func main() {
	//Pars args
	if err := parseArgs(); err != nil {
		log.Fatalf("%v\n", err)
		return
	}

	//Start servers
	if err := initCoAP(); err != nil {
		log.Fatalf("initCoAP error: %v\n", err)
		return
	}
	if err := initMQTT(); err != nil {
		log.Fatalf("initMQTT error: %v\n", err)
		return
	}

	//Interupt handling and wait exit
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Printf("Interupt[%s]: Finishing\n", sig)
		mqttConnection.Disconnect()
		done <- true
	}()

	<-done
}
