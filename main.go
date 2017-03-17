package main

import (
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/ndphu/espresso-commons"
	"github.com/ndphu/espresso-commons/model/device"
	"runtime"
	"strings"
	"time"
)

var (
	DefaultQos byte = 1
	//HealthReportTopic = "/es"
	TopicHealth   string        = "/esp/devices/health"
	DeviceType    string        = "fake"
	FakeSerial    string        = ""
	HealthTimeout time.Duration = 30
	StartTime     time.Time
)

func connectToBroker() (mqtt.Client, error) {
	o := mqtt.NewClientOptions()
	o.AddBroker("tcp://19november.freeddns.org:5370")
	o.SetClientID(fmt.Sprintf("fake-device-client-id-%s", FakeSerial))
	o.SetAutoReconnect(false)

	client := mqtt.NewClient(o)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	return client, nil
}

func generateHealthMessage() *device.DeviceStatus {
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)
	return &device.DeviceStatus{
		Serial: FakeSerial,
		Uptime: int(time.Since(StartTime).Nanoseconds() / 1000000),
		Free:   int(memStats.Frees),
	}
}

// Publish a health message to health topic
// The mesage should have device serial, following by the free heap space and uptime
// More information is not limited, base on device type
func publishHealth(client mqtt.Client) {
	hm := generateHealthMessage()
	data, err := json.Marshal(hm)
	if err != nil {
		panic(err)
	}

	if token := client.Publish(TopicHealth, DefaultQos, false, data); token.Wait() && token.Error() != nil {
		panic("Fail to publish hello message")
	}

	fmt.Println("Published health message")
}

func processMessage(c mqtt.Client, msg string) {
	parts := strings.Split(msg, ";")
	switch parts[0] {
	case "PING":
		publishHealth(c)
		break
	case "BLINK":
		fmt.Println("Blink onboard led", parts[1], "times with delay", parts[2], "ms")
		break
	case "GPIO_WRITE":
		fmt.Println("Will set GPIO pin", parts[1], "to", parts[2])
		break
	}
}

func main() {
	StartTime = time.Now()
	fmt.Println("Starting fake device")
	// create a fake serial
	FakeSerial = fmt.Sprintf("fake-%d", time.Now().Unix())
	fmt.Println("Serial:", FakeSerial)
	// connect to broker
	client, err := connectToBroker()
	if err != nil {
		panic(err)
	} else {
		fmt.Println("Connected to broker")
	}
	defer client.Disconnect(5000)

	commandTopic := commons.GetCommandTopicFromSerial(FakeSerial)
	if token := client.Subscribe(commandTopic, DefaultQos, func(c mqtt.Client, msg mqtt.Message) {
		//fmt.Println("...")
		processMessage(c, string(msg.Payload()))
	}); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	defer client.Unsubscribe(commandTopic)

	for {
		publishHealth(client)
		time.Sleep(HealthTimeout * time.Second)
	}

}
