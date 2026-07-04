package services

import (
	"ego/models"
	"ego/repositories"
)

// GetAllSettings returns all settings
func GetAllSettings() ([]models.Setting, error) {
	return repositories.GetAllSettings()
}

// GetSetting returns a setting by key
func GetSetting(key string) (*models.Setting, error) {
	return repositories.GetSetting(key)
}

// SetSetting sets a setting value
func SetSetting(key, value, description string) error {
	return repositories.SetSetting(key, value, description)
}
