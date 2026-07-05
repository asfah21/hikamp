package models

import "database/sql"

// Device represents a Hikvision device
type Device struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	IPAddress string         `json:"ip_address"`
	Port      int            `json:"port"`
	Username  string         `json:"username"`
	Password  string         `json:"password"`
	Location  string         `json:"location"`
	Status    string         `json:"status"` // online, offline
	Firmware  string         `json:"firmware"`
	LastSync  sql.NullString `json:"last_sync"`
	Enabled   bool           `json:"enabled"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}
