package repository

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sort"

	"zenthril-backend/internal/config"
)

type Shard struct {
	ID     string
	DSN    string
	Weight int
}

type ShardManager struct {
	shards []Shard
	ring   []ringNode
}

type ringNode struct {
	hash  uint64
	shard Shard
}

func NewShardManager(shards []Shard, virtualNodes int) (*ShardManager, error) {
	if len(shards) == 0 {
		return nil, errors.New("at least one shard is required")
	}
	if virtualNodes <= 0 {
		virtualNodes = 128
	}

	manager := &ShardManager{
		shards: append([]Shard(nil), shards...),
	}
	for _, shard := range shards {
		weight := shard.Weight
		if weight <= 0 {
			weight = 100
		}
		nodeCount := (virtualNodes * weight) / 100
		if nodeCount <= 0 {
			nodeCount = 1
		}
		for i := 0; i < nodeCount; i++ {
			key := fmt.Sprintf("%s:%d", shard.ID, i)
			manager.ring = append(manager.ring, ringNode{hash: hashKey(key), shard: shard})
		}
	}

	sort.Slice(manager.ring, func(i, j int) bool {
		return manager.ring[i].hash < manager.ring[j].hash
	})
	return manager, nil
}

func NewShardManagerFromConfig(cfg config.Config) (*ShardManager, error) {
	shards := make([]Shard, 0, len(cfg.Postgres.Shards))
	for _, shard := range cfg.Postgres.Shards {
		shards = append(shards, Shard{
			ID:     shard.ID,
			DSN:    shard.DSN,
			Weight: shard.Weight,
		})
	}
	return NewShardManager(shards, cfg.Sharding.VirtualNodes)
}

func (m *ShardManager) ShardForUserID(userID string) Shard {
	return m.pick("user:" + userID)
}

func (m *ShardManager) ShardForGuildID(guildID string) Shard {
	return m.pick("guild:" + guildID)
}

func (m *ShardManager) ShardForChannelID(channelID string) Shard {
	return m.pick("channel:" + channelID)
}

func (m *ShardManager) Shards() []Shard {
	return append([]Shard(nil), m.shards...)
}

func (m *ShardManager) pick(key string) Shard {
	if len(m.ring) == 0 {
		return Shard{}
	}
	h := hashKey(key)
	idx := sort.Search(len(m.ring), func(i int) bool {
		return m.ring[i].hash >= h
	})
	if idx == len(m.ring) {
		idx = 0
	}
	return m.ring[idx].shard
}

func hashKey(value string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(value))
	return h.Sum64()
}
