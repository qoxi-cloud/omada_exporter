package hamqtt

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Client retention: entities and device_trackers of clients that have not been
// seen for --mqtt-client-retention-hours are removed from MQTT (and therefore
// from Home Assistant).

func (p *Publisher) clientRetention() time.Duration {
	hours := p.client.Settings().MQTTClientRetentionHours
	if hours <= 0 {
		return 0
	}
	return time.Duration(hours) * time.Hour
}

// clientExpired reports whether a client last seen at lastSeen is older than
// the configured retention. Unknown lastSeen never expires.
func (p *Publisher) clientExpired(lastSeen, now time.Time) bool {
	retention := p.clientRetention()
	if retention <= 0 || lastSeen.IsZero() {
		return false
	}
	return now.Sub(lastSeen) > retention
}

// removeClientTracker clears a dynamic tracker that has been absent longer
// than the retention window (discovery, state, attributes and local memory).
func (p *Publisher) removeClientTracker(id string, tracker clientTracker) {
	discoveryPrefix := topicPrefix(p.client.Settings().MQTTDiscoveryPrefix)
	p.publishBytes(fmt.Sprintf("%s/device_tracker/%s/%s/config", discoveryPrefix, p.discoveryNode(), id), []byte{}, true)
	p.publishBytes(tracker.StateTopic, []byte{}, true)
	p.publishBytes(tracker.AttributesTopic, []byte{}, true)
	log.Info().Str("client", id).Msg("removed device tracker of a client absent longer than the retention window")
	p.mu.Lock()
	delete(p.knownClients, id)
	delete(p.lastClientTrackers, id)
	delete(p.clientLastSeen, id)
	delete(p.clientAttributes, tracker.AttributesTopic)
	p.mu.Unlock()
}

// expireAbsentClientEntities removes retained client sensors whose last
// update is older than the retention window and that were not republished in
// this cycle. Device and port entities are never touched here.
func (p *Publisher) expireAbsentClientEntities(current map[string]entity, knownDiscovery map[string]string, knownStates map[string]map[string]string) {
	retention := p.clientRetention()
	if retention <= 0 {
		return
	}
	now := time.Now().UTC()
	for discoveryTopic, stateTopic := range knownDiscovery {
		if _, isCurrent := current[discoveryTopic]; isCurrent {
			continue
		}
		if !p.ownsDiscovery(discoveryTopic) || !p.ownsEntityState(stateTopic) {
			continue
		}
		labels := knownStates[stateTopic]
		if labels == nil || labels["mac"] == "" || !strings.HasPrefix(labels["metric"], "net_client_") || !p.ownsLabels(labels) {
			continue
		}
		observed, err := time.Parse(time.RFC3339, labels["last_updated"])
		if err != nil || observed.IsZero() || now.Sub(observed) <= retention {
			continue
		}
		p.publishBytes(discoveryTopic, []byte{}, true)
		p.publishBytes(stateTopic, []byte{}, true)
		log.Info().Str("topic", discoveryTopic).Str("metric", labels["metric"]).Str("client", labels["mac"]).Msg("removed entity of a client absent longer than the retention window")
		p.mu.Lock()
		delete(p.retainedDiscovery, discoveryTopic)
		delete(p.retainedStates, stateTopic)
		delete(p.published, discoveryTopic)
		p.mu.Unlock()
	}
}
