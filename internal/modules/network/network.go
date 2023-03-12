package network

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/Wifx/gonetworkmanager"
	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"
	"github.com/umeshlumbhani/go-wifi-connect/internal/interfaces"
	"github.com/umeshlumbhani/go-wifi-connect/internal/models"
)

// Config represent this module
type Config struct {
	Log               *logrus.Logger
	NetworkManager    gonetworkmanager.NetworkManager
	CMD               interfaces.Command
	WifiDevice        gonetworkmanager.DeviceWireless
	HotSpotConnection gonetworkmanager.ActiveConnection
	isHotSpotCreated  bool
	Cfg               models.ConfigHandler
	WifiInterface     string
	AccessPoints      []AccessPoint
	HTTPServer        interfaces.HTTPServer
}

// AccessPoint represents Access point
type AccessPoint struct {
	SSID     string
	Path     dbus.ObjectPath
	Strength uint8
	Security models.SECURITY
}

// NewNetwork returns access to this module
func NewNetwork(l *logrus.Logger, cmd interfaces.Command, cfg models.ConfigHandler) (*Config, error) {
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		err = fmt.Errorf("found error on NewNetworkManager [%s]", err.Error())
		return nil, err
	}

	wDevice, err := getWifiDevice(nm)
	if err != nil {
		return nil, err
	}

	var dInterface string
	dInterface, err = wDevice.GetPropertyInterface()
	if err != nil {
		return nil, err
	}

	l.Info(fmt.Sprintf("device interface : %s", dInterface))
	return &Config{
		Log:            l,
		Cfg:            cfg,
		CMD:            cmd,
		NetworkManager: nm,
		WifiDevice:     wDevice,
		WifiInterface:  dInterface,
	}, nil
}

// StartPortal used to start wifi-connect captive portal
func (c *Config) StartPortal() {
	c.Log.Info("starting wifi connect captive portal")
	err := c.CreateHotSpot()
	if err != nil {
		panic(err)
	}
	c.HTTPServer.StartHTTPServer()
	return
}

// ClosePortal used to close wifi connect captive portal
func (c *Config) ClosePortal() {
	c.CloseHotSpot()
	c.Log.Info("closed hotspot")

	c.HTTPServer.CloseHTTPServer()
	c.Log.Info("closed HTTP Server")
}

// GetAccessPoint method used to get access point for the captive portal
func (c *Config) GetAccessPoint() (accessPoints []models.AccessPoint, err error) {
	if len(c.AccessPoints) > 0 {
		accessPoints = make([]models.AccessPoint, len(c.AccessPoints))
		for i, Ap := range c.AccessPoints {
			accessPoints[i] = models.AccessPoint{
				SSID:     Ap.SSID,
				Security: Ap.Security.String(),
			}
		}
	}
	return
}

// CreateHotSpot returns network manager info
func (c *Config) CreateHotSpot() (err error) {
	cfg := c.Cfg.Fetch()
	if c.isHotSpotCreated {
		err = errors.New("CreateHotSpot - Hotspot already created")
		c.Log.Error(err.Error())
		return
	}
	c.Log.Info("CreateHotSpot - creating access point")
	c.AccessPoints, err = c.getAccessPoint(10)
	if err != nil {
		c.Log.Error(fmt.Sprintf("found error on getWirelessDevice - getAccessPoint [%s]", err.Error()))
		return
	}
	connection := make(map[string]map[string]interface{})
	wl := map[string]interface{}{
		"ssid":     []byte(cfg.SSID),
		"band":     "bg",
		"hidden":   false,
		"mode":     "ap",
		"security": "802-11-wireless-security",
	}

	cn := map[string]interface{}{
		"autoconnect":    false,
		"id":             cfg.SSID,
		"interface-name": c.WifiInterface,
		"type":           "802-11-wireless",
	}

	ipv4 := make(map[string]interface{})
	ipv4["method"] = "manual"
	ipv4Address := make(map[string]interface{})
	ipv4Address["address"] = cfg.Gateway
	ipv4Address["prefix"] = uint32(24)
	ipv4AddressData := make([]map[string]interface{}, 1)
	ipv4AddressData[0] = ipv4Address
	ipv4["address-data"] = ipv4AddressData

	ipv6 := map[string]interface{}{
		"method": "ignore",
	}

	security := map[string]interface{}{
		"key-mgmt": "wpa-psk",
		"psk":      cfg.Passphrase,
	}

	connection["802-11-wireless"] = wl
	connection["802-11-wireless-security"] = security
	connection["connection"] = cn
	connection["ipv4"] = ipv4
	connection["ipv6"] = ipv6
	var hpConn gonetworkmanager.ActiveConnection
	hpConn, err = c.NetworkManager.AddAndActivateConnection(connection, c.WifiDevice)
	if err != nil {
		c.Log.Error(fmt.Sprintf("CreateHotSpot - found error on AddAndActivateConnection - %s", err.Error()))
		c.isHotSpotCreated = false
		c.HotSpotConnection = nil
		return
	}

	var isActivated bool
	isActivated, err = c.waitForConnectionState(20, hpConn, gonetworkmanager.NmActiveConnectionStateActivated)
	if err != nil {
		c.Log.Error(fmt.Sprintf("CreateHotSpot - found error on waitForConnectionState: %s", err.Error()))
		c.isHotSpotCreated = false
		c.HotSpotConnection = nil
		return
	}
	if isActivated {
		c.Log.Info(fmt.Sprintf("CreateHotSpot - Access point created - %s\n", cfg.SSID))
		c.isHotSpotCreated = true
		c.CMD.StartDnsmasq(c.WifiInterface)
		c.HotSpotConnection = hpConn
		return
	}

	err = errors.New("CreateHotSpot - connection could not be activated, closing connection")
	c.Log.Error(err.Error())
	var conn gonetworkmanager.Connection
	conn, err = hpConn.GetPropertyConnection()
	if err != nil {
		err = fmt.Errorf("CreateHotSpot - found error on GetPropertyConnection [%s]", err.Error())
		c.Log.Error(err.Error())
		return
	}
	err = c.NetworkManager.DeactivateConnection(hpConn)
	if err != nil {
		err = fmt.Errorf("CreateHotSpot - found error on DeactivateConnection [%s]", err.Error())
		c.Log.Error(err.Error())
		return
	}
	err = conn.Delete()
	if err != nil {
		err = fmt.Errorf("CreateHotSpot - found error on Delete [%s]", err.Error())
		c.Log.Error(err.Error())
		return
	}
	c.isHotSpotCreated = false
	c.HotSpotConnection = nil
	return
}

// CloseHotSpot used to close hotspot connection
func (c *Config) CloseHotSpot() (err error) {
	if c.isHotSpotCreated {
		c.Log.Info("Close access point")
		var conn gonetworkmanager.Connection
		conn, err = c.HotSpotConnection.GetPropertyConnection()
		if err != nil {
			err = fmt.Errorf("CloseHotSpot - found error on GetPropertyConnection [%s]", err.Error())
			c.Log.Error(err.Error())
			return
		}
		err = c.NetworkManager.DeactivateConnection(c.HotSpotConnection)
		if err != nil {
			err = fmt.Errorf("CloseHotSpot - found error on DeactivateConnection [%s]", err.Error())
			c.Log.Error(err.Error())
			return
		}
		err = conn.Delete()
		if err != nil {
			err = fmt.Errorf("CloseHotSpot - found error on Delete [%s]", err.Error())
			c.Log.Error(err.Error())
			return
		}
		c.CMD.KillDNSMasq()
		c.isHotSpotCreated = false
		c.HotSpotConnection = nil
		time.Sleep(5 * time.Second)
	}
	return
}

// Connect method used to connect to the network by captive portal
func (c *Config) Connect(ssid string, pwd string, identity string) (flg bool) {
	var err error
	flg = false
	err = c.deleteConnectionIfSameNetworkExists(ssid)
	if err != nil {
		c.Log.Error(err.Error())
		return
	}
	c.ClosePortal()
	c.Log.Info(fmt.Sprintf("connecting access point ---> %s", ssid))
	var ap gonetworkmanager.AccessPoint
	ap, err = c.getAccessPointFromSSID(ssid)
	if err != nil {
		c.Log.Error(err.Error())
		return
	}

	connection := make(map[string]map[string]interface{})
	connection["802-11-wireless"] = make(map[string]interface{})
	connection["802-11-wireless"]["security"] = "802-11-wireless-security"
	var cred map[string]map[string]interface{}
	cred, err = getWirelessCredentials(ap, pwd, identity)
	for k, val := range cred {
		connection[k] = val
	}
	var wifiConn gonetworkmanager.ActiveConnection
	wifiConn, err = c.NetworkManager.AddAndActivateWirelessConnection(connection, c.WifiDevice, ap)
	if err != nil {
		c.Log.Error(fmt.Sprintf("found error on AddAndActivateWirelessConnection: %s", err.Error()))
		return
	}
	var isActivated bool
	isActivated, err = c.waitForConnectionState(20, wifiConn, gonetworkmanager.NmActiveConnectionStateActivated)
	if err != nil {
		c.Log.Error(fmt.Sprintf("found error on waitForConnectionState: %s", err.Error()))
		return
	}
	if isActivated {
		var cFLag bool
		var connErr error
		cFLag, connErr = c.waitForConnectivity(20)
		if connErr != nil {
			c.Log.Warn(fmt.Sprintf("Getting Internet connectivity failed: %s", err.Error()))
		}
		if cFLag {
			c.Log.Info("Internet connectivity established")
		} else {
			c.Log.Warn("Cannot establish Internet connectivity")
		}
		flg = true
		return
	}
	var conn gonetworkmanager.Connection
	conn, err = wifiConn.GetPropertyConnection()
	if err != nil {
		c.Log.Error(fmt.Sprintf("found error on GetPropertyConnection of new created connection: %s", err.Error()))
		return
	}
	err = conn.Delete()
	if err != nil {
		c.Log.Error(fmt.Sprintf("found error on deleting connection object: %s", err.Error()))
		return
	}
	c.Log.Warn(fmt.Sprintf("Connection to access point not activated %s", ssid))
	c.StartPortal()
	return
}

func (c *Config) getAccessPoint(retryLimit int) (ap []AccessPoint, err error) {
	try := 0
start:
	var activeAPoints []gonetworkmanager.AccessPoint
	activeAPoints, err = c.WifiDevice.GetAccessPoints()
	if err != nil {
		return
	}
	var loopErr error
	var tempAP = make(map[string]AccessPoint)
	for _, aPoint := range activeAPoints {
		path := aPoint.GetPath()
		var ssid string
		var strength uint8
		ssid, loopErr = aPoint.GetPropertySSID()
		if loopErr != nil {
			c.Log.Error(fmt.Sprintf("getAccessPoint - found error on GetPropertySSID - %s", loopErr.Error()))
			continue
		}
		strength, loopErr = aPoint.GetPropertyStrength()
		if loopErr != nil {
			c.Log.Error(fmt.Sprintf("getAccessPoint - found error on GetPropertyStrength - %s", loopErr.Error()))
			continue
		}
		var security models.SECURITY
		security, loopErr = getAccessPointSecurity(aPoint)
		if loopErr != nil {
			c.Log.Error(fmt.Sprintf("getAccessPoint - found error on getAccessPointSecurity - %s", loopErr.Error()))
			continue
		}
		if ssid != "" {
			a := AccessPoint{
				SSID:     ssid,
				Security: security,
				Strength: strength,
				Path:     path,
			}
			tempAP[ssid] = a
		}
	}
	for _, a := range tempAP {
		ap = append(ap, a)
	}
	if len(ap) == 0 {
		if try >= retryLimit {
			err = errors.New("no accesspoint found")
		}
		try = try + 1
		time.Sleep(2 * time.Second)
		goto start
	}
	sort.Slice(ap[:], func(i, j int) bool {
		return ap[i].Strength > ap[j].Strength
	})
	var printAccessPoint []string

	for _, a := range ap {
		printAccessPoint = append(printAccessPoint, a.SSID)
	}
	return
}

func getWirelessCredentials(ap gonetworkmanager.AccessPoint, pwd string, identity string) (security80211 map[string]map[string]interface{}, err error) {
	security80211 = make(map[string]map[string]interface{})
	var security models.SECURITY
	security, err = getAccessPointSecurity(ap)
	if err != nil {
		return
	}
	if (security & models.ENTERPRISE) == models.ENTERPRISE {
		setting1 := make(map[string]interface{})
		setting2 := make(map[string]interface{})
		setting1["key-mgmt"] = "wpa-eap"
		setting2["eap"] = []string{"peap"}
		setting2["identity"] = identity
		setting2["password"] = pwd
		setting2["phase2-auth"] = "mschapv2"
		security80211["802-11-wireless-security"] = setting1
		security80211["802-1x"] = setting2
	} else if (security&models.WPA2) == models.WPA2 || (security&models.WPA) == models.WPA {
		setting1 := make(map[string]interface{})
		setting1["key-mgmt"] = "wpa-psk"
		setting1["psk"] = pwd
		security80211["802-11-wireless-security"] = setting1
	} else if (security & models.WEP) == models.WEP {
		setting1 := make(map[string]interface{})
		setting1["wep-key-type"] = uint32(2)
		setting1["wep-key0"] = pwd
		security80211["802-11-wireless-security"] = setting1
	}
	return
}

func getAccessPointSecurity(ap gonetworkmanager.AccessPoint) (security models.SECURITY, err error) {
	var flag, wpaFlag, rsnFlag uint32

	flag, err = ap.GetPropertyFlags()
	if err != nil {
		err = fmt.Errorf("found error on getting flags")
		return
	}
	wpaFlag, err = ap.GetPropertyWPAFlags()
	if err != nil {
		err = fmt.Errorf("found error on getting flags")
		return
	}
	rsnFlag, err = ap.GetPropertyRSNFlags()
	if err != nil {
		err = fmt.Errorf("found error on getting flags")
		return
	}
	security = models.NONE
	if (flag&models.ApFlagsPrivacy.U32()) == models.ApFlagsPrivacy.U32() && wpaFlag == models.ApSecNone.U32() && rsnFlag == models.ApSecNone.U32() {
		security += models.WEP
	}

	if wpaFlag != models.ApSecNone.U32() {
		security += models.WPA
	}

	if rsnFlag != models.ApSecNone.U32() {
		security += models.WPA2
	}

	if (wpaFlag&models.ApSecKeyMgmt8021X.U32()) == models.ApSecKeyMgmt8021X.U32() || (rsnFlag&models.ApSecKeyMgmt8021X.U32()) == models.ApSecKeyMgmt8021X.U32() {
		security += models.ENTERPRISE
	}
	return
}

func (c *Config) waitForConnectivity(timeout int) (bool, error) {
	var totalTime = 0
	for {
		nmConn, err := c.NetworkManager.GetPropertyConnectivity()
		if err != nil {
			return false, err
		}
		if nmConn == gonetworkmanager.NmConnectivityFull || nmConn == gonetworkmanager.NmConnectivityLimited {
			c.Log.Debug(fmt.Sprintf("Connectivity established: %s / %ds elapsed", nmConn.String(), totalTime))
			return true, nil
		} else if totalTime >= timeout {
			c.Log.Debug(fmt.Sprintf("Timeout reached in waiting for connectivity: %s / %ds elapsed", nmConn.String(), totalTime))
			return false, nil
		}
		totalTime = totalTime + 1
		time.Sleep(1 * time.Second)
		c.Log.Debug(fmt.Sprintf("Still waiting for connectivity: %s / %ds elapsed", nmConn.String(), totalTime))
	}
}

func (c *Config) waitForConnectionState(timeout int, ac gonetworkmanager.ActiveConnection, cs gonetworkmanager.NmActiveConnectionState) (bool, error) {
	var totalTime = 0
	for {
		state, err := ac.GetPropertyState()
		if err != nil {
			return false, err
		}
		if state == cs {
			c.Log.Debug(fmt.Sprintf("connection state matched: %s / %ds elapsed", state.String(), totalTime))
			return true, nil
		} else if totalTime >= timeout {
			c.Log.Debug(fmt.Sprintf("Timeout reached in waiting for connection state: %s / %ds elapsed", cs.String(), totalTime))
			return false, nil
		}
		totalTime = totalTime + 1
		time.Sleep(1 * time.Second)
		c.Log.Debug(fmt.Sprintf("Still waiting for connection state: %s, required %s / %ds elapsed", state.String(), cs.String(), totalTime))
	}
}

func (c *Config) deleteConnectionIfSameNetworkExists(ssid string) (err error) {
	c.Log.Info("deleting existing connection of same network")
	conns, err := c.WifiDevice.GetPropertyAvailableConnections()
	if err != nil {
		err = fmt.Errorf("found error on GetPropertyAvailableConnections - %s", err.Error())
		return
	}
	for _, conn := range conns {
		var sett gonetworkmanager.ConnectionSettings
		sett, err = conn.GetSettings()
		if err != nil {
			err = fmt.Errorf("found error on GetSettings - %s", err.Error())
			return
		}
		if _, ok := sett["802-11-wireless"]; ok {
			if connSsid, ok2 := sett["802-11-wireless"]["ssid"].([]byte); ok2 {
				if ssid == string(connSsid) {
					c.Log.Info("connection exists, deleted.")
					err = conn.Delete()
					if err != nil {
						err = fmt.Errorf("deleteConnectionIfSameNetworkExists - found error on Delete - %s", err.Error())
						return
					}
				}
			}
		}
	}
	return
}

func getWifiDevice(nm gonetworkmanager.NetworkManager) (d gonetworkmanager.DeviceWireless, err error) {
	devices, err := nm.GetAllDevices()
	if err != nil {
		err = fmt.Errorf("found error on GetAllDevices [%s]", err.Error())
		return
	}

	for _, device := range devices {
		var dType gonetworkmanager.NmDeviceType
		dType, err = device.GetPropertyDeviceType()
		if err != nil {
			err = fmt.Errorf("found error on GetPropertyDeviceType [%s]", err.Error())
			return
		}
		path := device.GetPath()
		switch dType {
		case gonetworkmanager.NmDeviceTypeWifi:
			var state gonetworkmanager.NmDeviceState
			state, err = device.GetPropertyState()
			if err != nil {
				err = fmt.Errorf("found error on GetPropertyState [%s]", err.Error())
				return
			}
			if state != gonetworkmanager.NmDeviceStateUnmanaged {
				d, err = gonetworkmanager.NewDeviceWireless(path)
				if err != nil {
					err = fmt.Errorf("found error on getWirelessDevice - NewDeviceWireless [%s]", err.Error())
					return
				}
				return
			}
		}
	}
	err = errors.New("could not find wifi device")
	return
}

func (c *Config) getAccessPointFromSSID(ssid string) (accessPoint gonetworkmanager.AccessPoint, err error) {
	var aPoints []AccessPoint
	aPoints, err = c.getAccessPoint(10)
	if err != nil {
		err = fmt.Errorf("found error on GetAccessPoints: %s", err.Error())
		return
	}
	fmt.Println(aPoints)
	for _, aPoint := range aPoints {
		if aPoint.SSID == ssid {
			accessPoint, err = gonetworkmanager.NewAccessPoint(aPoint.Path)
			return
		}
	}
	err = fmt.Errorf("could not found accesspoint with ssid: %s", ssid)
	return
}
