// Package hub — zone auth token (ERA_FEDERATED_ZONE_KEY).
package hub

import (
	"crypto/subtle"
	"os"
)

// ZoneKeyConfigured — true если задан ERA_FEDERATED_ZONE_KEY.
func ZoneKeyConfigured() bool {
	return os.Getenv("ERA_FEDERATED_ZONE_KEY") != ""
}

// ValidateZoneToken проверяет Bearer/X-ERA-Zone-Token против ERA_FEDERATED_ZONE_KEY.
// Пустой ключ в env — dev-режим (пропуск).
func ValidateZoneToken(provided string) bool {
	expected := os.Getenv("ERA_FEDERATED_ZONE_KEY")
	if expected == "" {
		return true
	}
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
