package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/surgemq/message"
	"github.com/surgemq/surgemq/service"
)

var (
	mqttConnection = service.Client{}
)

const (
	gatewayPostfixToClient   = "to"
	gatewayPostfixFromClient = "from"
	gatewayPrefix            = "gateway"

	letterArray = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0987654321"
)

func getRandString(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	res := make([]byte, n)
	for i := range res {
		res[i] = letterArray[r.Intn(len(letterArray))]
	}

	return string(res)
}

func addrToString(a *net.UDPAddr) string {
	return a.IP.String() + ":" + strconv.Itoa(a.Port)
}

func getMQTTTopicNameTo(a *net.UDPAddr) string {
	return gatewayPrefix + "/" + addrToString(a) + "/" + gatewayPostfixToClient
}

func getMQTTTopicNameFrom(a *net.UDPAddr) string {
	return gatewayPrefix + "/" + addrToString(a) + "/" + gatewayPostfixFromClient
}

func parseTopic(topicName string) (addr *net.UDPAddr, err error) {
	splitTopic := strings.Split(topicName, "/")
	netString := splitTopic[len(splitTopic)-2]

	addr, err = net.ResolveUDPAddr("udp", netString)
	if err != nil {
		return nil, err
	}

	return addr, nil
}

func onMQTTMessageToServer(msg *message.PublishMessage) error {
	topicName := string(msg.Topic())
	log.Printf("onMQTTMessageToServer: new message was published in '%s' \n", topicName)

	if err := checkCoAPPayload(msg.Payload()); err != nil {
		log.Printf("onMQTTMessageToServer: Error on parsing udp packet: %v\n", err)
		return err
	}

	clientAddr, err := parseTopic(topicName)
	if err != nil {
		log.Printf("Error while parsing topic: %v \n", err)
		return err
	}

	gatewayClient := getClientByRealAddr(clientAddr)

	if gatewayClient == nil {
		gatewayClient, err = initNewCoApConnectionToServer(clientAddr)
		if err != nil {
			log.Printf("Error while dialing new connection to server: %v\n", err)
			return err
		}
	}

	if err := sendCOAPMessageToServer(gatewayClient.con, msg.Payload()); err != nil {
		log.Printf("Error while sendCOAPMessage to server: %v\n", err)
		return err
	}

	return nil
}

func onMQTTMessageToClient(msg *message.PublishMessage) error {
	topicName := string(msg.Topic())
	log.Printf("onMQTTMessageToClient: new message was published in '%s' \n", topicName)

	if err := checkCoAPPayload(msg.Payload()); err != nil {
		log.Printf("onMQTTMessageToClient: Error on parsing udp packet: %v\n", err)
		return err
	}

	clientAddr, err := parseTopic(topicName)
	if err != nil {
		log.Printf("Error while parsing topic: %v \n", err)
		return err
	}

	if err := sendCOAPMessageToClient(clientAddr, msg.Payload()); err != nil {
		log.Printf("Error while sendCOAPMessage to client: %v\n", err)
		return err
	}

	return nil
}

func subscribeToMQTTTopic(topicName string, handler func(*message.PublishMessage) error) error {
	submsg := message.NewSubscribeMessage()
	submsg.AddTopic([]byte(topicName), 0)

	return mqttConnection.Subscribe(submsg, nil, handler)
}

func publishMQTTMessage(topicName string, payload []byte) error {
	pubmsg := message.NewPublishMessage()
	pubmsg.SetTopic([]byte(topicName))
	pubmsg.SetPayload(payload)

	return mqttConnection.Publish(pubmsg, nil)
}

func initMQTTConnection() error {
	mqttConnectionString := fmt.Sprintf("tcp://%s:%d", mqttIP, mqttPort)

	clientID := "gateway_" + getRandString(10)
	msg := message.NewConnectMessage()
	msg.SetWillQos(1)
	msg.SetVersion(4)
	msg.SetCleanSession(true)
	msg.SetClientId([]byte(clientID))
	msg.SetKeepAlive(5)
	msg.SetWillTopic([]byte("coap_gateway"))
	msg.SetWillMessage([]byte(clientID))

	err := mqttConnection.Connect(mqttConnectionString, msg)
	if err != nil {
		log.Printf("initMQTTConnection: connection error: %v\n", err)
		return err
	}

	log.Printf("initMQTTConnection success [id: %s]\n", clientID)

	return nil
}

func initMQTT() error {
	if err := initMQTTConnection(); err != nil {
		return err
	}

	if isClient {
		if err := subscribeToMQTTTopic(gatewayPrefix+"/+/"+gatewayPostfixToClient, onMQTTMessageToClient); err != nil {
			return err
		}
	} else if isServer {
		if err := subscribeToMQTTTopic(gatewayPrefix+"/+/"+gatewayPostfixFromClient, onMQTTMessageToServer); err != nil {
			return err
		}

	}

	return nil
}
