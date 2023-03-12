package interfaces

import "github.com/umeshlumbhani/go-wifi-connect/internal/models"

// Network represents network module
type Network interface {
	GetAccessPoint() (accessPoints []models.AccessPoint, err error)
	CreateHotSpot() (err error)
	CloseHotSpot() (err error)
	Connect(ssid string, pwd string, identity string) (flg bool)
}
