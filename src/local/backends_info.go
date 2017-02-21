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
