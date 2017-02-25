package local

import (
	"net"
	"sync"

	"common"
	"config"
	"outbound"
	"outbound/ss"
)

type BackendsInformation []*BackendInfo

func (slice BackendsInformation) Len() int {
	return len(slice)
}

func (slice BackendsInformation) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type ByLastSecondBps struct{ BackendsInformation }

func (slice ByLastSecondBps) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetLastSecondBps() > statistics.StatisticMap[slice.BackendsInformation[j]].GetLastSecondBps()
}

type ByLastMinuteBps struct{ BackendsInformation }

func (slice ByLastMinuteBps) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetLastMinuteBps() > statistics.StatisticMap[slice.BackendsInformation[j]].GetLastMinuteBps()
}

type ByLastTenMinutesBps struct{ BackendsInformation }

func (slice ByLastTenMinutesBps) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetLastTenMinutesBps() > statistics.StatisticMap[slice.BackendsInformation[j]].GetLastTenMinutesBps()
}

type ByLastHourBps struct{ BackendsInformation }

func (slice ByLastHourBps) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetLastHourBps() > statistics.StatisticMap[slice.BackendsInformation[j]].GetLastHourBps()
}

type ByHighestLastSecondBps struct{ BackendsInformation }

func (slice ByHighestLastSecondBps) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastSecondBps() > statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastSecondBps()
}

type ByHighestLastMinuteBps struct{ BackendsInformation }

func (slice ByHighestLastMinuteBps) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastMinuteBps() > statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastMinuteBps()
}

type ByHighestLastTenMinutesBps struct{ BackendsInformation }

func (slice ByHighestLastTenMinutesBps) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastTenMinutesBps() > statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastTenMinutesBps()
}

type ByHighestLastHourBps struct{ BackendsInformation }

func (slice ByHighestLastHourBps) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastHourBps() > statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastHourBps()
}

type ByLatency struct{ BackendsInformation }

func (slice ByLatency) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	if statistics.StatisticMap[slice.BackendsInformation[i]].GetLatency() == 0 {
		return false
	}
	if statistics.StatisticMap[slice.BackendsInformation[j]].GetLatency() == 0 {
		return true
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetLatency() < statistics.StatisticMap[slice.BackendsInformation[j]].GetLatency()
}

type ByFailedCount struct{ BackendsInformation }

func (slice ByFailedCount) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetFailedCount() < statistics.StatisticMap[slice.BackendsInformation[j]].GetFailedCount()
}

type ByTotalUpload struct{ BackendsInformation }

func (slice ByTotalUpload) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetTotalUploaded() > statistics.StatisticMap[slice.BackendsInformation[j]].GetTotalUploaded()
}

type ByTotalDownload struct{ BackendsInformation }

func (slice ByTotalDownload) Less(i, j int) bool {
	statistics.RLock()
	defer statistics.RUnlock()
	if bi, ok := statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return statistics.StatisticMap[slice.BackendsInformation[i]].GetTotalDownload() < statistics.StatisticMap[slice.BackendsInformation[j]].GetTotalDownload()
}

type BackendsInformationWrapper struct {
	sync.RWMutex
	BackendsInformation
}

func NewBackendsInformationWrapper() *BackendsInformationWrapper {
	biw := &BackendsInformationWrapper{}
	biw.BackendsInformation = make(BackendsInformation, 0)
	return biw
}

func (biw *BackendsInformationWrapper) Append(bi *BackendInfo) {
	biw.Lock()
	defer biw.Unlock()
	biw.BackendsInformation = append(biw.BackendsInformation, bi)
}

func (biw *BackendsInformationWrapper) Remove(i int) {
	biw.Lock()
	defer biw.Unlock()
	biw.BackendsInformation = append(biw.BackendsInformation[:i], biw.BackendsInformation[i+1:]...)
}

func (biw *BackendsInformationWrapper) Get(i int) *BackendInfo {
	biw.RLock()
	defer biw.RUnlock()
	if i >= len(biw.BackendsInformation) {
		return nil
	}
	return biw.BackendsInformation[i]
}

func (biw *BackendsInformationWrapper) Len() int {
	biw.RLock()
	defer biw.RUnlock()
	return len(biw.BackendsInformation)
}

var (
	// backends collection contains remote server information
	backends = NewBackendsInformationWrapper()
)

func removeDeprecatedServers() {
	// remove the ones that is not included in new config
	for i := 0; i < backends.Len(); {
		backendInfo := backends.Get(i)
		find := false
		for _, s := range config.Configurations.OutboundsConfig {
			if backendInfo.address == s.Address {
				find = true
				break
			}
		}

		if !find {
			// remove this element from backends array
			statistics.Delete(backends.Get(i))
			backends.Remove(i)
			i = 0
		} else {
			i++
		}
	}
}

func updateExistsOutboundConfig(outboundConfig *outbound.Outbound) bool {
	for _, backendInfo := range backends.BackendsInformation {
		if backendInfo.address == outboundConfig.Address {
			backendInfo.protocolType = outboundConfig.Type
			backendInfo.encryptMethod = outboundConfig.Method
			backendInfo.encryptPassword = outboundConfig.Key
			if outboundConfig.Timeout != 0 {
				backendInfo.timeout = outboundConfig.Timeout
			} else {
				backendInfo.timeout = config.Configurations.Generals.Timeout
			}

			return true
		}
	}
	return false
}

func addNewOutboundConfig(outboundConfig *outbound.Outbound) {
	// append directly
	backendInfo := &BackendInfo{
		id:           common.GenerateRandomString(4),
		address:      outboundConfig.Address,
		protocolType: outboundConfig.Type,
		SSRInfo: SSRInfo{
			obfs:          outboundConfig.Obfs,
			obfsParam:     outboundConfig.ObfsParam,
			protocol:      outboundConfig.Protocol,
			protocolParam: outboundConfig.ProtocolParam,
			SSInfo: SSInfo{
				encryptMethod:   outboundConfig.Method,
				encryptPassword: outboundConfig.Key,
				tcpFastOpen:     outboundConfig.TCPFastOpen,
			},
		},

		HTTPSProxyInfo: HTTPSProxyInfo{
			insecureSkipVerify: outboundConfig.TLSInsecureSkipVerify,
			domain:             outboundConfig.TLSDomain,
			CommonProxyInfo: CommonProxyInfo{
				username: outboundConfig.Username,
				password: outboundConfig.Password,
			},
		},
	}
	if outboundConfig.Timeout != 0 {
		backendInfo.timeout = outboundConfig.Timeout
	} else {
		backendInfo.timeout = config.Configurations.Generals.Timeout
	}
	backendInfo.local = outboundConfig.Local

	backends.Append(backendInfo)

	stat := common.NewStatistic()
	statistics.Insert(backendInfo, stat)
}

func updateNewServers() {
	// add or update the ones that is included in the config
	for _, outboundConfig := range config.Configurations.OutboundsConfig {
		if outboundConfig.Type == "shadowsocks" || outboundConfig.Type == "ss" {
			_, err := ss.NewStreamCipher(outboundConfig.Method, outboundConfig.Key)
			if err != nil {
				common.Error("Failed generating ciphers:", err)
				continue
			}
		}

		// don't append directly, scan the existing elements and update them
		if !updateExistsOutboundConfig(outboundConfig) {
			addNewOutboundConfig(outboundConfig)
		}

		if len(config.DefaultKey) == 0 {
			config.DefaultKey = outboundConfig.Key
		}
		if len(config.DefaultPort) == 0 {
			_, config.DefaultPort, _ = net.SplitHostPort(outboundConfig.Address)
		}
		if len(config.DefaultMethod) == 0 {
			config.DefaultMethod = outboundConfig.Method
		}
	}
}
