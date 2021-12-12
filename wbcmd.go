package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Device struct {
	WB_MQTT_HOST string
	NAME         string
	TARGET       string
	CHANNEL      string
}

type Command struct {
	target      string
	action      string
	device_name string
}

var ALLOWED_ACTIONS []string = []string{"up", "down"}
var EXTRA_ACTIONS_FOR_POWER []string = []string{"restart"}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func publish(client mqtt.Client, channel string, action string) {
	fmt.Printf("Pubblish " + action + " to channel " + channel)
	token := client.Publish(channel, 0, false, action)
	token.Wait()
	time.Sleep(time.Second)
}

// func subscribe(client mqtt.Client, channel string) {
// 	token := client.Subscribe(channel, 1, nil)
// 	token.Wait()
// }

func get_mqtt_client(url string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(url)
	opts.SetClientID("go_mqtt_client")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	return client
}

func execute_command(command Command, devices []Device) error {
	device, _ := get_device_for_command(devices, command)

	client := get_mqtt_client(device.WB_MQTT_HOST)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	if command.action == "up" {
		publish(client, device.CHANNEL, "1")
	} else if command.action == "down" {
		publish(client, device.CHANNEL, "0")
	} else if command.action == "restart" {
		publish(client, device.CHANNEL, "0")
		time.Sleep(7 * time.Second)
		publish(client, device.CHANNEL, "1")
	} else {
		return errors.New("Action " + command.action + " is unknown")
	}

	client.Disconnect(250)
	return nil
}

func parse_json_config(file_data []byte) ([]Device, error) {
	var devices []Device
	err := json.Unmarshal(file_data, &devices)
	if err != nil {
		return nil, err
	}

	// convertnames and targets to lower case
	for _, device := range devices {
		device.NAME = strings.ToLower(device.NAME)
		device.TARGET = strings.ToLower(device.TARGET)
	}

	// check parsed devices
	for _, device := range devices {
		if device.WB_MQTT_HOST == "" ||
			device.NAME == "" ||
			device.TARGET == "" ||
			device.CHANNEL == "" {
			return nil, errors.New("incorrect device detected")
		}
	}
	return devices, nil
}

func read_stand_config(config_path string) []Device {
	// Open our jsonFile
	jsonFile, err := os.Open(config_path)
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	devices, err := parse_json_config([]byte(byteValue))
	if err != nil {
		log.Fatal("Can't parce config: " + err.Error())
	}

	for i, device := range devices {
		devices[i].NAME = strings.ToLower(device.NAME)
		devices[i].TARGET = strings.ToLower(device.TARGET)
	}

	return devices
}

func parce_cmd_args(devices []Device) (Command, error) {
	if len(os.Args) == 2 && is_help_request(os.Args[1]) {
		fmt.Print(generate_help_page(devices))
		os.Exit(0)
	} else if len(os.Args) != 4 {
		return Command{}, errors.New("expectes 3 arguments")
	}
	command := Command{strings.ToLower(os.Args[1]), strings.ToLower(os.Args[2]), strings.ToLower(os.Args[3])}

	is_correct, err := is_command_correct(command, devices)
	if err != nil {
		return Command{}, err
	}
	if !is_correct {
		return Command{}, errors.New("unkown error")
	}

	return command, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func is_command_correct(command Command, devices []Device) (bool, error) {
	// check device correctness
	devices_by_name := get_devices_by_name(devices, command.device_name)
	if len(devices_by_name) == 0 {
		return false, errors.New("Device '" + command.device_name + "' is not defined")
	}
	// check target for selected device
	_, err := get_device_for_command(devices, command)
	if err != nil {
		return false, err
	}
	// check is action correct
	if command.target == "power" {
		if !contains(append(ALLOWED_ACTIONS, EXTRA_ACTIONS_FOR_POWER...), command.action) {
			return false, errors.New("Action '" + command.action + "' is not supported")
		}
	} else {
		if !contains(ALLOWED_ACTIONS, command.action) {
			return false, errors.New("Action '" + command.action + "' is not supported")
		}
	}
	return true, nil
}

func get_devices_by_name(devices []Device, name string) []Device {
	var selected_devices []Device
	for _, device := range devices {
		if device.NAME == name {
			selected_devices = append(selected_devices, device)
		}
	}
	return selected_devices
}

func get_devices_for_command(devices []Device, command Command) []Device {
	var selected_devices []Device
	for _, device := range devices {
		if device.NAME == command.device_name && device.TARGET == command.target {
			selected_devices = append(selected_devices, device)
		}
	}
	return selected_devices
}

func get_device_for_command(devices []Device, command Command) (Device, error) {
	selected_devices := get_devices_for_command(devices, command)
	if len(selected_devices) == 0 {
		return Device{}, errors.New("no applicable device")
	}
	return selected_devices[0], nil
}

func is_help_request(cmd_argument string) bool {
	cmd_argument = strings.ToLower(cmd_argument)
	if cmd_argument == "-h" || cmd_argument == "--help" || cmd_argument == "?" {
		return true
	}
	return false
}

func get_basic_help_page() string {
	var basic_help_page string
	basic_help_page += "\nUsage:\n"
	basic_help_page += "\twbcmd <target> <action> <device>\n"
	basic_help_page += "\nOptions:\n"
	return basic_help_page
}

func generate_help_page(devices []Device) string {
	allowed_targets := []string{}
	allowed_devices := []string{}
	allowed_actions := []string{"up", "down"}

	is_contains := func(list []string, item string) bool {
		for _, i := range list {
			if i == item {
				return true
			}
		}
		return false
	}

	for _, device := range devices {
		target := strings.ToLower(device.TARGET)
		if !is_contains(allowed_targets, target) {
			allowed_targets = append(allowed_targets, target)
		}
		device := strings.ToLower(device.NAME)
		if !is_contains(allowed_devices, device) {
			allowed_devices = append(allowed_devices, device)
		}

		if target == "power" && !is_contains(allowed_actions, "restart") {
			allowed_actions = append(allowed_actions, "restart")
		}
	}

	slice_to_string := func(s []string) string {
		sort.Strings(s)
		var my_str string
		is_first := true
		for _, item := range s {
			if !is_first {
				my_str += ", "
			}
			my_str += "'" + item + "'"
			is_first = false
		}
		return my_str
	}

	help_page := get_basic_help_page()
	help_page += "\t<target>\t\t\tWhat you want switch. Allowed values: " + slice_to_string(allowed_targets) + "\n"
	help_page += "\t<action>\t\t\tWhat action you want to do. Allowed values: " + slice_to_string(allowed_actions)
	if is_contains(allowed_actions, "restart") {
		help_page += " ('restart' may be applicable for 'power' only)"
	}
	help_page += "\n"
	help_page += "\t<device>\t\t\tDevice name. Allowed values: " + slice_to_string(allowed_devices)
	help_page += "\n"
	return help_page
}

func get_mqtt_env_config_path() string {
	path, ok := os.LookupEnv("MQTT_ENV_CONFIG")
	if !ok {
		return "/etc/wbcmd/mqtt_env_config"
	} else {
		return path
	}
}

func main() {
	devices := read_stand_config(get_mqtt_env_config_path())
	command, err := parce_cmd_args(devices)
	if err != nil {
		fmt.Printf("Command is incorrect: " + err.Error())
		os.Exit(1)
	}

	err = execute_command(command, devices)
	if err != nil {
		fmt.Printf("Error! Command is not executed: " + err.Error() + "\n")
		os.Exit(2)
	}
}
