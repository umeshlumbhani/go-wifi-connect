package command

import (
	"bufio"
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/umeshlumbhani/go-wifi-connect/internal/models"
)

// Command for device network commands.
type Command struct {
	Log      *logrus.Logger
	Commands map[string]*exec.Cmd
	Cfg      models.ConfigHandler
}

// NewCommand returns access to this module
func NewCommand(l *logrus.Logger, cfg models.ConfigHandler) *Command {
	return &Command{
		Commands: make(map[string]*exec.Cmd),
		Cfg:      cfg,
		Log:      l,
	}
}

func (c *Command) processMonitor(id string, cmd *exec.Cmd) {
	c.Log.Info(fmt.Sprintf("ProcessMonitor got with id - %s", id))

	cmdStdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		c.Log.Error(fmt.Sprintf("ProcessMonitor - found error on StdoutPipe %s", err.Error()))
		panic(err)
	}

	cmdStderrReader, err := cmd.StderrPipe()
	if err != nil {
		c.Log.Error(fmt.Sprintf("ProcessMonitor - found error on StderrPipe %s", err.Error()))
		panic(err)
	}

	stdOutScanner := bufio.NewScanner(cmdStdoutReader)
	go func() {
		for stdOutScanner.Scan() {
			c.Log.Info(fmt.Sprintf("ProcessMonitor - Output for [%s|%s] ===> %s", id, cmd.Path, stdOutScanner.Text()))
		}
	}()

	stdErrScanner := bufio.NewScanner(cmdStderrReader)
	go func() {
		for stdErrScanner.Scan() {
			c.Log.Error("command", "ProcessMonitor", fmt.Sprintf("ProcessMonitor - Output for [%s|%s] ===> %s", id, cmd.Path, stdErrScanner.Text()))
		}
	}()

	err = cmd.Run()
	if err != nil {
		c.Log.Error(fmt.Sprintf("ProcessMonitor - found error on cmd.run : %s", err.Error()))
	}
	if _, ok := c.Commands[id]; ok {
		delete(c.Commands, id)
		c.Log.Info(fmt.Sprintf("ProcessMonitor - [%s] exit.", id))
	}
}

// StartDnsmasq starts dnsmasq.
func (c *Command) StartDnsmasq(dInt string) {
	c.Log.Info("Start DNS masq")
	cfg := c.Cfg.Fetch()

	// hostapd is enabled, fire up dnsmasq
	args := []string{
		fmt.Sprintf("--address=/#/%s", cfg.Gateway), // Don't read the hostnames in /etc/hosts.
		fmt.Sprintf("--dhcp-range=%s", cfg.DHCPRange),
		fmt.Sprintf("--dhcp-option=option:router,%s", cfg.Gateway),
		fmt.Sprintf("--interface=%s", dInt),
		"--keep-in-foreground",
		"--bind-interfaces",
		"--except-interface=lo",
		"--conf-file",
		"--no-hosts",
	}

	cmd := exec.Command("dnsmasq", args...)
	// add command to the commands map TODO close the readers
	c.Commands["dnsmasq"] = cmd
	go c.processMonitor("dnsmasq", cmd)
}

// KillDNSMasq used to kill dnsmasq
func (c *Command) KillDNSMasq() {
	if _, ok := c.Commands["dnsmasq"]; ok {
		c.Log.Info("Kill DNS masq")
		cmd := c.Commands["dnsmasq"]
		err := cmd.Process.Kill()
		if err != nil {
			c.Log.Error(fmt.Sprintf("KillDNSMasq - found error on cmd.Process.Kill : %s", err.Error()))
		}
	}
	return
}
