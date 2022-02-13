package main

import (
	"strconv"
	"strings"
	"testing"
)

func TestParseOneDevice(t *testing.T) {
	one_device_json := `[
		{
			"WB_MQTT_HOST": "wiren-board.test-stand",
			"NAME": "DEVICE1",
			"TARGET": "POWER",
			"CHANNEL": "/devices/wb-mr3_17/controls/K1/on"
		}
		]
		`

	devices, _ := parse_json_config([]byte(one_device_json))
	if len(devices) != 1 {
		t.Error("Expected len 1")
	}

	device := devices[0]
	if device.WB_MQTT_HOST != "wiren-board.test-stand" ||
		device.NAME != "DEVICE1" ||
		device.TARGET != "POWER" ||
		device.CHANNEL != "/devices/wb-mr3_17/controls/K1/on" {
		t.Error("Data parsed incorrectly")
	}
}

func TestParseTwoDevices(t *testing.T) {
	two_devices_json := `[
		{
			"WB_MQTT_HOST": "wiren-board.test-stand",
			"NAME": "DEVICE1",
			"TARGET": "POWER",
			"CHANNEL": "/devices/wb-mr3_17/controls/K1/on"
		},
		{
			"WB_MQTT_HOST": "wiren-board.test-stand",
			"NAME": "DEVICE2",
			"TARGET": "BOOT",
			"CHANNEL": "/devices/wb-mr3_17/controls/K2/on"
		}
		]
		`

	devices, _ := parse_json_config([]byte(two_devices_json))
	if len(devices) != 2 {
		t.Error("Expected len 1")
	}

	for _, device := range devices {
		if device.NAME == "DEVICE1" {
			if device.WB_MQTT_HOST != "wiren-board.test-stand" ||
				device.NAME != "DEVICE1" ||
				device.TARGET != "POWER" ||
				device.CHANNEL != "/devices/wb-mr3_17/controls/K1/on" {
				t.Error("Data parsed incorrectly")
			}
		} else if device.NAME == "DEVICE2" {
			if device.WB_MQTT_HOST != "wiren-board.test-stand" ||
				device.NAME != "DEVICE2" ||
				device.TARGET != "BOOT" ||
				device.CHANNEL != "/devices/wb-mr3_17/controls/K2/on" {
				t.Error("Data parsed incorrectly")
			}
		}
	}
}

func TestParseError(t *testing.T) {
	one_device_json := `[
		{
			"WB_MQTT_HOST": "wiren-board.test-stand",
			"NAME": "DEVICE1",
			"TARGET": "POWER"
		}
		]
		`

	_, err := parse_json_config([]byte(one_device_json))
	if err == nil {
		t.Error("Expected error")
	}
}

func TestHelpPageForOneDevice(t *testing.T) {
	one_device_json := `[
		{
			"WB_MQTT_HOST": "wiren-board.test-stand",
			"NAME": "DEVICE1",
			"TARGET": "POWER",
			"CHANNEL": "/devices/wb-mr3_17/controls/K1/on"
		}
		]
		`
	expected_help_page := `
Usage:
	wbcmd <target> <action> <device>

Options:
	<target>			What you want switch. Allowed values: 'power'
	<action>			What action you want to do. Allowed values: 'down', 'restart', 'up' ('restart' may be applicable for 'power' only)
	<device>			Device name. Allowed values: 'device1'
`

	devices, _ := parse_json_config([]byte(one_device_json))
	help_page := generate_help_page(devices)
	if help_page != expected_help_page {
		t.Error("Generated help page is different with expected:\nGenerated:\n" + help_page + "Expected:\n" + expected_help_page)
	}
}

func TestIsCommandCorrect(t *testing.T) {
	device1 := Device{"host", "d1", "power", "K1"}
	device2 := Device{"host", "d2", "boot_mode", "K2"}
	devices := []Device{device1, device2}

	// Positive scenarios
	{
		command := Command{"power", "up", "d1"}
		is_correct, err := is_command_correct(command, devices)
		if !is_correct {
			t.Error("Command {" + command.target + " " + command.action + " " + command.device_name + "} is correct. But: " + err.Error())
		}
	}

	{
		command := Command{"power", "down", "d1"}
		is_correct, _ := is_command_correct(command, devices)
		if !is_correct {
			t.Error("Command {" + command.target + " " + command.action + " " + command.device_name + "} is correct")
		}
	}

	{
		command := Command{"power", "restart", "d1"}
		is_correct, _ := is_command_correct(command, devices)
		if !is_correct {
			t.Error("Command {" + command.target + " " + command.action + " " + command.device_name + "} is correct")
		}
	}

	{
		command := Command{"boot_mode", "up", "d2"}
		is_correct, _ := is_command_correct(command, devices)
		if !is_correct {
			t.Error("Command {" + command.target + " " + command.action + " " + command.device_name + "} is correct")
		}
	}

	{
		command := Command{"boot_mode", "down", "d2"}
		is_correct, _ := is_command_correct(command, devices)
		if !is_correct {
			t.Error("Command {" + command.target + " " + command.action + " " + command.device_name + "} is correct")
		}
	}

	// Negative scenarios
	{
		command := Command{"boot_mode", "down", "d1"}
		is_correct, err := is_command_correct(command, devices)
		if is_correct {
			t.Error("Command {" + command.target + " " + command.action + " " + command.device_name + "} is incorrect. 'boot_mode' is not applicable for d1")
		}
		if err == nil {
			t.Error("Error should not be nil.")
		}
	}

	{
		command := Command{"power", "skip", "d1"}
		is_correct, err := is_command_correct(command, devices)
		if is_correct {
			t.Error("Command {" + command.target + " " + command.action + " " + command.device_name + "} is incorrect. Action 'skip' is not applicable")
		}
		if err == nil {
			t.Error("Error should not be nil.")
		}
	}

	{
		command := Command{"boot_mode", "restart", "d2"}
		is_correct, err := is_command_correct(command, devices)
		if is_correct {
			t.Error("Command {" + command.target + " " + command.action + " " + command.device_name + "} is incorrect. Action 'restart' is not applicable for target 'boot_mode'")
		}
		if err == nil {
			t.Error("Error should not be nil.")
		}
	}

	{
		command := Command{"boot_mode", "down", "d3"}
		is_correct, err := is_command_correct(command, devices)
		if is_correct {
			t.Error("Command {" + command.target + " " + command.action + " " + command.device_name + "} is incorrect. Device d3 is not defined")
		}
		if err == nil {
			t.Error("Error should not be nil.")
		}
	}
}

func TestGetDevicesByName(t *testing.T) {
	device1 := Device{"host", "d1", "POWER", "K1"}
	device2 := Device{"host", "d1", "BOOT_MODE", "K2"}
	device3 := Device{"host", "d2", "BOOT_MODE", "K2"}
	devices := []Device{device1, device2, device3}

	d1_devices := get_devices_by_name(devices, "d1")
	if len(d1_devices) != 2 {
		t.Error("2 devices is expected, but received " + strconv.Itoa(len(d1_devices)))
	}

	expected_devices := []Device{device1, device2}
	for _, device := range d1_devices {
		for i, expected_device := range expected_devices {
			if device == expected_device {
				expected_devices = remove_element_from_list(expected_devices, i)
			}
		}
	}

	if len(expected_devices) != 0 {
		t.Error("Returned list of devices is not expected")
	}
}

func TestGetDeviceForCommand(t *testing.T) {
	device1 := Device{"host", "d1", "power", "K1"}
	device2 := Device{"host", "d1", "boot_mode", "K2"}
	device3 := Device{"host", "d2", "boot_mode", "K2"}
	devices := []Device{device1, device2, device3}

	{
		command := Command{"boot_mode", "down", "d1"}

		device, err := get_device_for_command(devices, command)
		if device.NAME != command.device_name || strings.ToLower(device.TARGET) != command.target {
			t.Error("Incorrect device selected: " + device.NAME + ", " + device.TARGET)
		}
		if err != nil {
			t.Error("Error is not expecte: " + err.Error())
		}
	}

	{
		command := Command{"boot_mode", "down", "d3"}

		device, err := get_device_for_command(devices, command)
		if err == nil || device.TARGET != "" {
			t.Error("Device should not be found. But found: " + device.NAME + ", " + device.TARGET)
		}
	}
}

func TestExecuteCommandWithoutChecking(t *testing.T) {
	device1 := Device{"tcp://127.0.0.1:1883", "d1", "power", "K1"}
	device2 := Device{"tcp://127.0.0.1:1883", "d1", "boot_mode", "K2"}
	device3 := Device{"tcp://127.0.0.1:1883", "d2", "boot_mode", "K2"}
	devices := []Device{device1, device2, device3}

	{
		command := Command{"boot_mode", "down", "d1"}

		err := execute_command(command, devices)
		if err != nil {
			t.Error("Error is not expecte: " + err.Error())
		}
	}
}

func remove_element_from_list(s []Device, i int) []Device {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
