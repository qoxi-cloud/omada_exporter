package hamqtt

import (
	"testing"

	"github.com/RCooLeR/omada_exporter/internal/config"
	"github.com/RCooLeR/omada_exporter/internal/omada/session"
)

func TestPurgeLegacyEntitiesTombstonesOnlyLegacyPrefixedRecords(t *testing.T) {
	mqttClient := &recordingMQTTClient{messages: map[string][]byte{}}
	publisher := &Publisher{
		client:                 &session.Client{Config: &config.Config{MQTTRetain: true, MQTTDiscoveryPrefix: "homeassistant", MQTTTopicPrefix: "omada_exporter"}},
		mqtt:                   mqttClient,
		published:              map[string]struct{}{},
		retainedDiscovery:      map[string]string{},
		retainedStates:         map[string]map[string]string{},
		legacyTrackerDiscovery: map[string]struct{}{},
		retainedLoaded:         true,
	}
	legacyDiscovery := "homeassistant/sensor/omada_exporter/omada_device_cpu_x/config"
	legacyState := "omada_exporter/entities/omada_device_cpu_x/state"
	currentDiscovery := "homeassistant/sensor/omada_exporter/net_device_cpu_x/config"
	currentState := "omada_exporter/entities/net_device_cpu_x/state"
	trackerDiscovery := "homeassistant/device_tracker/omada_exporter/aabbccddeeff/config"
	publisher.retainedDiscovery[legacyDiscovery] = legacyState
	publisher.retainedStates[legacyState] = map[string]string{"metric": "omada_device_cpu_percentage", "device_mac": "AA"}
	publisher.retainedDiscovery[currentDiscovery] = currentState
	publisher.retainedStates[currentState] = map[string]string{"metric": "net_device_cpu_percentage", "device_mac": "AA"}
	publisher.legacyTrackerDiscovery[trackerDiscovery] = struct{}{}

	publisher.purgeLegacyEntities()

	for _, topic := range []string{legacyDiscovery, legacyState, trackerDiscovery} {
		if payload, ok := mqttClient.messages[topic]; !ok || len(payload) != 0 {
			t.Fatalf("%s was not tombstoned: present=%v payload=%q", topic, ok, payload)
		}
	}
	if _, ok := mqttClient.messages[currentDiscovery]; ok {
		t.Fatalf("current net_ entity must not be touched")
	}
	if _, ok := publisher.retainedDiscovery[legacyDiscovery]; ok {
		t.Fatalf("legacy discovery should be forgotten after purge")
	}
}
