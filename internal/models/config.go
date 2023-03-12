package models

import (
	"flag"
	"fmt"
)

type ConfigHandler interface {
	Fetch() Config
}

const (
	defaultGateway         string = "192.168.42.1"
	defaultDHCPRange       string = "192.168.42.2,192.168.42.254"
	defaultSSID            string = "WiFi Connect"
	defaultActivityTimeout int    = 0
	defaultUIDirectory     string = "ui"
	defaultListeningPort   string = "80"
)

type Config struct {
	Gateway         string
	Port            string
	DHCPRange       string
	SSID            string
	Interface       string
	Passphrase      string
	UIDirectory     string
	ActivityTimeout int
}

// SetConfig used to set configuration from cli argument
func NewConfig() *Config {
	var winterface, gateway, dhcprange, ssid, uidir, port, pwd string
	var at int

	flag.StringVar(&winterface, "portal-interface", "", "Wireless network interface to be used by WiFi Connect")
	flag.StringVar(&ssid, "portal-ssid", defaultSSID, fmt.Sprintf("SSID of the captive portal WiFi network (default: %s)", defaultSSID))
	flag.StringVar(&pwd, "portal-passphrase", "", "WPA2 Passphrase of the captive portal WiFi network (default: none)")
	flag.StringVar(&gateway, "portal-gateway", defaultGateway, fmt.Sprintf("Gateway of the captive portal WiFi network (default: %s)", defaultGateway))
	flag.StringVar(&dhcprange, "portal-dhcp-range", defaultDHCPRange, fmt.Sprintf("DHCP range of the WiFi network (default: %s)", defaultDHCPRange))
	flag.StringVar(&port, "portal-listening-port", defaultListeningPort, fmt.Sprintf("Listening port of the captive portal web server (default: %s)", defaultListeningPort))
	flag.IntVar(&at, "activity-timeout", defaultActivityTimeout, "Exit if no activity for the specified time (seconds) (default: 0)")
	flag.StringVar(&uidir, "ui-directory", defaultUIDirectory, fmt.Sprintf("Web UI directory location (default: %s)", defaultUIDirectory))

	flag.Parse()

	return &Config{
		Gateway:         gateway,
		Port:            port,
		DHCPRange:       dhcprange,
		SSID:            ssid,
		Interface:       winterface,
		Passphrase:      pwd,
		UIDirectory:     uidir,
		ActivityTimeout: at,
	}
}

func (c *Config) Fetch() Config {
	return *c
}
