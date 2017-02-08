package local

import "sync"

type BackendsInformation []*BackendInfo

func (slice BackendsInformation) Len() int {
	return len(slice)
}

func (slice BackendsInformation) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

type ByLastSecondBps struct{ BackendsInformation }

func (slice ByLastSecondBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLastSecondBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetLastSecondBps()
}

type ByLastMinuteBps struct{ BackendsInformation }

func (slice ByLastMinuteBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLastMinuteBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetLastMinuteBps()
}

type ByLastTenMinutesBps struct{ BackendsInformation }

func (slice ByLastTenMinutesBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLastTenMinutesBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetLastTenMinutesBps()
}

type ByLastHourBps struct{ BackendsInformation }

func (slice ByLastHourBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLastHourBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetLastHourBps()
}

type ByHighestLastSecondBps struct{ BackendsInformation }

func (slice ByHighestLastSecondBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastSecondBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastSecondBps()
}

type ByHighestLastMinuteBps struct{ BackendsInformation }

func (slice ByHighestLastMinuteBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastMinuteBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastMinuteBps()
}

type ByHighestLastTenMinutesBps struct{ BackendsInformation }

func (slice ByHighestLastTenMinutesBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastTenMinutesBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastTenMinutesBps()
}

type ByHighestLastHourBps struct{ BackendsInformation }

func (slice ByHighestLastHourBps) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetHighestLastHourBps() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetHighestLastHourBps()
}

type ByLatency struct{ BackendsInformation }

func (slice ByLatency) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	if Statistics.StatisticMap[slice.BackendsInformation[i]].GetLatency() == 0 {
		return false
	}
	if Statistics.StatisticMap[slice.BackendsInformation[j]].GetLatency() == 0 {
		return true
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetLatency() < Statistics.StatisticMap[slice.BackendsInformation[j]].GetLatency()
}

type ByFailedCount struct{ BackendsInformation }

func (slice ByFailedCount) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetFailedCount() < Statistics.StatisticMap[slice.BackendsInformation[j]].GetFailedCount()
}

type ByTotalUpload struct{ BackendsInformation }

func (slice ByTotalUpload) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetTotalUploaded() > Statistics.StatisticMap[slice.BackendsInformation[j]].GetTotalUploaded()
}

type ByTotalDownload struct{ BackendsInformation }

func (slice ByTotalDownload) Less(i, j int) bool {
	Statistics.RLock()
	defer Statistics.RUnlock()
	if bi, ok := Statistics.StatisticMap[slice.BackendsInformation[i]]; !ok || bi == nil {
		return false
	}
	if sj, ok := Statistics.StatisticMap[slice.BackendsInformation[j]]; !ok || sj == nil {
		return false
	}
	return Statistics.StatisticMap[slice.BackendsInformation[i]].GetTotalDownload() < Statistics.StatisticMap[slice.BackendsInformation[j]].GetTotalDownload()
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
