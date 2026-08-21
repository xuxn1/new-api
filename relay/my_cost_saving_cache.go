package relay

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/mycostsaving"
	"github.com/samber/hot"
)

const myCostSavingCacheNamespace = "new-api:my_cost_saving:exact:v1"

type myCostSavingCacheEntry struct {
	Response dto.OpenAITextResponse `json:"response"`
}

type myCostSavingCacheCodec struct{}

func (c myCostSavingCacheCodec) Encode(v myCostSavingCacheEntry) (string, error) {
	data, err := common.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c myCostSavingCacheCodec) Decode(s string) (myCostSavingCacheEntry, error) {
	var v myCostSavingCacheEntry
	if err := common.UnmarshalJsonStr(s, &v); err != nil {
		return myCostSavingCacheEntry{}, err
	}
	return v, nil
}

var myCostSavingExactCache = cachex.NewHybridCache[myCostSavingCacheEntry](cachex.HybridCacheConfig[myCostSavingCacheEntry]{
	Namespace:  cachex.Namespace(myCostSavingCacheNamespace),
	Redis:      common.RDB,
	RedisCodec: myCostSavingCacheCodec{},
	RedisEnabled: func() bool {
		return common.RedisEnabled && common.RDB != nil
	},
	Memory: func() *hot.HotCache[string, myCostSavingCacheEntry] {
		return hot.NewHotCache[string, myCostSavingCacheEntry](hot.LRU, 10000).
			WithTTL(time.Duration(mycostsaving.GetSettings().ExactCacheTTLSeconds) * time.Second).
			WithJanitor().
			Build()
	},
})

func myCostSavingCacheTTL(match mycostsaving.Match) time.Duration {
	if !match.CacheEnabled || match.CacheTTLSeconds <= 0 {
		return 0
	}
	return time.Duration(match.CacheTTLSeconds) * time.Second
}

func myCostSavingCacheKey(info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest, match mycostsaving.Match, executionModel string) (string, error) {
	if info == nil || request == nil {
		return "", nil
	}
	cacheRequest, err := common.DeepCopy(request)
	if err != nil {
		return "", err
	}
	cacheRequest.SetModelName(info.OriginModelName)
	cacheRequest.Stream = nil
	cacheRequest.StreamOptions = nil

	scopeID := info.UsingGroup
	if match.CacheScope == "user" {
		scopeID = fmt.Sprintf("user:%d", info.UserId)
	}
	payload := struct {
		Group          string                    `json:"group"`
		Scope          string                    `json:"scope"`
		ScopeID        string                    `json:"scope_id"`
		OriginalModel  string                    `json:"original_model"`
		ExecutionModel string                    `json:"execution_model"`
		RequestPath    string                    `json:"request_path"`
		Request        dto.GeneralOpenAIRequest `json:"request"`
	}{
		Group:          info.UsingGroup,
		Scope:          match.CacheScope,
		ScopeID:        scopeID,
		OriginalModel:  info.OriginModelName,
		ExecutionModel: executionModel,
		RequestPath:    info.RequestURLPath,
		Request:        *cacheRequest,
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
