package hamqtt

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Metric names used to carry the controller vendor as prefix (omada_*). They
// are now vendor-neutral (net_*). Anything still retained under an old prefix
// is removed once at startup so Home Assistant does not keep duplicate,
// never-updated entities around.
// built by concatenation so a blanket vendor-prefix rename never rewrites it
var legacyMetricPrefixes = []string{"omada" + "_"}

func isLegacyMetric(name string) bool {
	for _, prefix := range legacyMetricPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func isLegacyUniqueID(uniqueID string) bool {
	return strings.HasPrefix(uniqueID, "omada"+"_client_") || strings.HasPrefix(uniqueID, "omada"+"_exporter_")
}

// purgeLegacyEntities clears retained discovery/state records that still use a
// legacy metric prefix, and device_tracker configs with a legacy unique_id, so
// the following publishAll recreates everything under the net_* names with
// clean entity ids (HA processes the removals before the new configs).
func (p *Publisher) purgeLegacyEntities() {
	if !p.client.Settings().MQTTRetain {
		return
	}
	p.mu.Lock()
	if !p.retainedLoaded {
		p.mu.Unlock()
		return
	}
	type victim struct{ discovery, state, metric string }
	victims := []victim{}
	for discoveryTopic, stateTopic := range p.retainedDiscovery {
		labels := p.retainedStates[stateTopic]
		metric := ""
		if labels != nil {
			metric = labels["metric"]
		}
		if metric != "" && !isLegacyMetric(metric) {
			continue
		}
		if metric == "" && !strings.Contains(discoveryTopic, "/") {
			continue
		}
		victims = append(victims, victim{discoveryTopic, stateTopic, metric})
	}
	trackers := make([]string, 0, len(p.legacyTrackerDiscovery))
	for topic := range p.legacyTrackerDiscovery {
		trackers = append(trackers, topic)
	}
	p.mu.Unlock()
	log.Info().Int("legacy_entities", len(victims)).Int("legacy_trackers", len(trackers)).Msg("legacy-prefixed retained records to purge")

	for _, v := range victims {
		if v.metric == "" {
			// discovery without a readable retained state: leave it to supersession
			continue
		}
		p.publishBytes(v.discovery, []byte{}, true)
		p.publishBytes(v.state, []byte{}, true)
		log.Info().Str("topic", v.discovery).Str("metric", v.metric).Msg("removed legacy-prefixed retained mqtt entity")
		p.mu.Lock()
		delete(p.retainedDiscovery, v.discovery)
		delete(p.retainedStates, v.state)
		delete(p.published, v.discovery)
		p.mu.Unlock()
	}
	for _, topic := range trackers {
		p.publishBytes(topic, []byte{}, true)
		log.Info().Str("topic", topic).Msg("removed legacy device_tracker discovery; it is re-announced with the net_ unique_id")
		p.mu.Lock()
		delete(p.legacyTrackerDiscovery, topic)
		delete(p.published, topic)
		p.mu.Unlock()
	}
}

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
