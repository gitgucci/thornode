package observer

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"gitlab.com/thorchain/thornode/bifrost/metrics"
	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	"gitlab.com/thorchain/thornode/common"
)

type PubKeyValidator interface {
	IsValidPoolAddress(addr string, chain common.Chain) (bool, common.ChainPoolInfo)
	HasPubKey(pk common.PubKey) bool
	AddPubKey(pk common.PubKey, _ bool)
	RemovePubKey(pk common.PubKey)
	GetSignPubKeys() common.PubKeys
	GetPubKeys() common.PubKeys
	Start() error
	Stop() error
}

type PK struct {
	PubKey common.PubKey
	Signer bool
}

// PubKeyManager it manages a list of pubkeys
type PubKeyManager struct {
	cdc        *codec.Codec
	pubkeys    []PK
	rwMutex    *sync.RWMutex
	logger     zerolog.Logger
	chainHost  string // thorchain host
	errCounter *prometheus.CounterVec
	m          *metrics.Metrics
	stopChan   chan struct{}
}

// NewPubKeyManager create a new instance of PubKeyManager
func NewPubKeyManager(chainHost string, m *metrics.Metrics) (*PubKeyManager, error) {
	return &PubKeyManager{
		cdc:        thorclient.MakeCodec(),
		logger:     log.With().Str("module", "thorchain_bridge").Logger(),
		chainHost:  chainHost,
		errCounter: m.GetCounterVec(metrics.PubKeyManagerError),
		m:          m,
		stopChan:   make(chan struct{}),
		rwMutex:    &sync.RWMutex{},
	}, nil
}

// Start to poll pubkeys from thorchain
func (pkm *PubKeyManager) Start() error {
	pubkeys, err := pkm.getPubkeys()
	if err != nil {
		return errors.Wrap(err, "fail to get pubkeys from thorchain")
	}
	for _, pk := range pubkeys {
		pkm.AddPubKey(pk, false)
	}
	go pkm.updatePubKeys()
	return nil
}

// Stop pubkey manager
func (pkm *PubKeyManager) Stop() error {
	defer pkm.logger.Info().Msg("pubkey manager stopped")
	close(pkm.stopChan)
	return nil
}

func (pkm *PubKeyManager) GetPubKeys() common.PubKeys {
	pubkeys := make(common.PubKeys, len(pkm.pubkeys))
	for i, pk := range pkm.pubkeys {
		pubkeys[i] = pk.PubKey
	}
	return pubkeys
}

func (pkm *PubKeyManager) GetSignPubKeys() common.PubKeys {
	pubkeys := make(common.PubKeys, 0)
	for _, pk := range pkm.pubkeys {
		if pk.Signer {
			pubkeys = append(pubkeys, pk.PubKey)
		}
	}
	return pubkeys
}

func (pkm *PubKeyManager) HasPubKey(pk common.PubKey) bool {
	for _, pubkey := range pkm.pubkeys {
		if pk.Equals(pubkey.PubKey) {
			return true
		}
	}
	return false
}

func (pkm *PubKeyManager) AddPubKey(pk common.PubKey, signer bool) {
	pkm.rwMutex.Lock()
	defer pkm.rwMutex.Unlock()

	if pkm.HasPubKey(pk) {
		// pubkey already exists, update the signer... but only if signer is true
		if signer {
			for i, pubkey := range pkm.pubkeys {
				if pk.Equals(pubkey.PubKey) {
					pkm.pubkeys[i].Signer = signer
				}
			}
		}
	} else {
		// pubkey doesn't exist yet, append it...
		pkm.pubkeys = append(pkm.pubkeys, PK{
			PubKey: pk,
			Signer: signer,
		})
	}
}

func (pkm *PubKeyManager) RemovePubKey(pk common.PubKey) {
	pkm.rwMutex.Lock()
	defer pkm.rwMutex.Unlock()
	for i, pubkey := range pkm.pubkeys {
		if pk.Equals(pubkey.PubKey) {
			pkm.pubkeys[i] = pkm.pubkeys[len(pkm.pubkeys)-1] // Copy last element to index i.
			pkm.pubkeys[len(pkm.pubkeys)-1] = PK{}           // Erase last element (write zero value).
			pkm.pubkeys = pkm.pubkeys[:len(pkm.pubkeys)-1]   // Truncate slice.
			break
		}
	}
}

func (pkm *PubKeyManager) updatePubKeys() {
	pkm.logger.Info().Msg("start to update pub keys")
	defer pkm.logger.Info().Msg("stop to update pub keys")
	for {
		select {
		case <-pkm.stopChan:
			return
		case <-time.After(time.Minute):
			pubkeys, err := pkm.getPubkeys()
			if nil != err {
				pkm.logger.Error().Err(err).Msg("fail to get pubkeys from thorchain")
			}
			for _, pk := range pubkeys {
				pkm.AddPubKey(pk, false)
			}
		}
	}
}

func matchAddress(addr string, chain common.Chain, key common.PubKey) (bool, common.ChainPoolInfo) {
	cpi, err := common.NewChainPoolInfo(chain, key)
	if err != nil {
		return false, common.EmptyChainPoolInfo
	}
	if strings.EqualFold(cpi.PoolAddress.String(), addr) {
		return true, cpi
	}
	return false, common.EmptyChainPoolInfo
}

// IsValidPoolAddress check whether the given address is a pool addr
func (pkm *PubKeyManager) IsValidPoolAddress(addr string, chain common.Chain) (bool, common.ChainPoolInfo) {
	pkm.rwMutex.RLock()
	defer pkm.rwMutex.RUnlock()

	for _, pk := range pkm.pubkeys {
		ok, cpi := matchAddress(addr, chain, pk.PubKey)
		if ok {
			return ok, cpi
		}
	}
	return false, common.EmptyChainPoolInfo
}

// getPubkeys from thorchain
func (pkm *PubKeyManager) getPubkeys() (common.PubKeys, error) {
	uri := url.URL{
		Scheme: "http",
		Host:   pkm.chainHost,
		Path:   "/thorchain/vaults/pubkeys",
	}
	resp, err := retryablehttp.Get(uri.String())
	if nil != err {
		return nil, errors.Wrap(err, "fail to get pubkeys from thorchain")
	}
	defer func() {
		if err := resp.Body.Close(); nil != err {
			pkm.logger.Error().Err(err).Msg("fail to close response body")
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(err, "fail to get pubkeys from thorchain")
	}

	var pubs struct {
		Asgard    common.PubKeys `json:"asgard"`
		Yggdrasil common.PubKeys `json:"yggdrasil"`
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return nil, errors.Wrap(err, "fail to read response body")
	}
	if err := pkm.cdc.UnmarshalJSON(buf, &pubs); nil != err {
		pkm.errCounter.WithLabelValues("fail_unmarshal_pubkeys", "").Inc()
		return nil, errors.Wrap(err, "fail to unmarshal pubkeys")
	}

	pubkeys := append(pubs.Asgard, pubs.Yggdrasil...)
	return pubkeys, nil
}