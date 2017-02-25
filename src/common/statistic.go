package common

import (
	"math"
	"sync"
	"sync/atomic"
)

type totalStat struct {
	totalUpload   uint64
	totalDownload uint64
}

type highestStat struct {
	highestLastSecondBps     uint64
	highestLastMinuteBps     uint64
	highestLastTenMinutesBps uint64
	highestLastHourBps       uint64
}

type lastStat struct {
	lastSecondBps     uint64
	lastMinuteBps     uint64
	lastTenMinutesBps uint64
	lastHourBps       uint64
}
type Statistic struct {
	sync.RWMutex
	totalStat
	highestStat
	lastStat
	latency            int64
	bytesCurrentSecond uint64
	bytesPerSecond     [3600]uint64
	failedCount        uint32
}

func NewStatistic() *Statistic {
	return &Statistic{
		latency: math.MaxInt64,
	}
}

func (s *Statistic) Tick() {
	b := atomic.SwapUint64(&s.bytesCurrentSecond, 0)
	s.Lock()
	// left shift
	copy(s.bytesPerSecond[:], append(s.bytesPerSecond[1:], b))
	s.Unlock()
	atomic.StoreUint64(&s.lastHourBps, s.bpsLastHour())
	if atomic.LoadUint64(&s.lastHourBps) > atomic.LoadUint64(&s.highestLastHourBps) {
		atomic.StoreUint64(&s.highestLastHourBps, atomic.LoadUint64(&s.lastHourBps))
	}
	atomic.StoreUint64(&s.lastTenMinutesBps, s.bpsLastTenMinutes())
	if atomic.LoadUint64(&s.lastTenMinutesBps) > atomic.LoadUint64(&s.highestLastTenMinutesBps) {
		atomic.StoreUint64(&s.highestLastTenMinutesBps, atomic.LoadUint64(&s.lastTenMinutesBps))
	}
	atomic.StoreUint64(&s.lastMinuteBps, s.bpsLastMinute())
	if atomic.LoadUint64(&s.lastMinuteBps) > atomic.LoadUint64(&s.highestLastMinuteBps) {
		atomic.StoreUint64(&s.highestLastMinuteBps, atomic.LoadUint64(&s.lastMinuteBps))
	}
	atomic.StoreUint64(&s.lastSecondBps, s.bpsLastSecond())
	if atomic.LoadUint64(&s.lastSecondBps) > atomic.LoadUint64(&s.highestLastSecondBps) {
		atomic.StoreUint64(&s.highestLastSecondBps, atomic.LoadUint64(&s.lastSecondBps))
	}
}

func (s *Statistic) bpsLastHour() uint64 {
	s.RLock()
	defer s.RUnlock()
	var sum uint64 = 0
	for _, b := range s.bytesPerSecond {
		sum += b
	}
	return sum / 3600
}

func (s *Statistic) GetLastHourBps() uint64 {
	return atomic.LoadUint64(&s.lastHourBps)
}

func (s *Statistic) SetLastHourBps(b uint64) {
	atomic.StoreUint64(&s.lastHourBps, b)
}

func (s *Statistic) GetHighestLastHourBps() uint64 {
	return atomic.LoadUint64(&s.highestLastHourBps)
}

func (s *Statistic) SetHighestLastHourBps(b uint64) {
	atomic.StoreUint64(&s.highestLastHourBps, b)
}
func (s *Statistic) bpsLastTenMinutes() uint64 {
	s.RLock()
	defer s.RUnlock()
	var sum uint64 = 0
	for i := 0; i < 600; i++ {
		sum += s.bytesPerSecond[3600-i-1]
	}
	return sum / 600
}

func (s *Statistic) GetLastTenMinutesBps() uint64 {
	return atomic.LoadUint64(&s.lastTenMinutesBps)
}

func (s *Statistic) SetLastTenMinutesBps(b uint64) {
	atomic.StoreUint64(&s.lastTenMinutesBps, b)
}

func (s *Statistic) GetHighestLastTenMinutesBps() uint64 {
	return atomic.LoadUint64(&s.highestLastTenMinutesBps)
}

func (s *Statistic) SetHighestLastTenMinutesBps(b uint64) {
	atomic.StoreUint64(&s.highestLastTenMinutesBps, b)
}
func (s *Statistic) bpsLastMinute() uint64 {
	s.RLock()
	defer s.RUnlock()
	var sum uint64 = 0
	for i := 0; i < 60; i++ {
		sum += s.bytesPerSecond[3600-i-1]
	}
	return sum / 60
}

func (s *Statistic) GetLastMinuteBps() uint64 {
	return atomic.LoadUint64(&s.lastMinuteBps)
}

func (s *Statistic) SetLastMinuteBps(b uint64) {
	atomic.StoreUint64(&s.lastMinuteBps, b)
}

func (s *Statistic) GetHighestLastMinuteBps() uint64 {
	return atomic.LoadUint64(&s.highestLastMinuteBps)
}

func (s *Statistic) SetHighestLastMinuteBps(b uint64) {
	atomic.StoreUint64(&s.highestLastMinuteBps, b)
}

func (s *Statistic) bpsLastSecond() uint64 {
	s.RLock()
	defer s.RUnlock()
	return s.bytesPerSecond[3599]
}

func (s *Statistic) GetLastSecondBps() uint64 {
	return atomic.LoadUint64(&s.lastSecondBps)
}

func (s *Statistic) SetLastSecondBps(b uint64) {
	atomic.StoreUint64(&s.lastSecondBps, b)
}

func (s *Statistic) GetHighestLastSecondBps() uint64 {
	return atomic.LoadUint64(&s.highestLastSecondBps)
}

func (s *Statistic) SetHighestLastSecondBps(b uint64) {
	atomic.StoreUint64(&s.highestLastSecondBps, b)
}

func (s *Statistic) BytesDownload(b uint64) {
	atomic.AddUint64(&s.totalDownload, b)
	atomic.AddUint64(&s.bytesCurrentSecond, b)
	TotalStat.AddDownload(b)
	DeltaStat.AddDownload(b)
}

func (s *Statistic) IncreaseFailedCount() {
	atomic.AddUint32(&s.failedCount, 1)
}

func (s *Statistic) ClearFailedCount() {
	atomic.StoreUint32(&s.failedCount, 0)
}

func (s *Statistic) SetFailedCount(b uint32) {
	atomic.StoreUint32(&s.failedCount, b)
}

func (s *Statistic) GetFailedCount() uint32 {
	return atomic.LoadUint32(&s.failedCount)
}

func (s *Statistic) ClearLatency() {
	atomic.StoreInt64(&s.latency, 0)
}

func (s *Statistic) GetLatency() int64 {
	return atomic.LoadInt64(&s.latency)
}

func (s *Statistic) SetLatency(l int64) {
	atomic.StoreInt64(&s.latency, l)
}

func (s *Statistic) IncreaseTotalUpload(b uint64) {
	atomic.AddUint64(&s.totalUpload, b)
	TotalStat.AddUpload(b)
	DeltaStat.AddUpload(b)
}

func (s *Statistic) ClearUpload() {
	atomic.StoreUint64(&s.totalUpload, 0)
}

func (s *Statistic) GetTotalUploaded() uint64 {
	return atomic.LoadUint64(&s.totalUpload)
}

func (s *Statistic) SetTotalUploaded(b uint64) {
	atomic.StoreUint64(&s.totalUpload, b)
}

func (s *Statistic) IncreaseTotalDownload(b uint64) {
	atomic.AddUint64(&s.totalDownload, b)
}

func (s *Statistic) ClearDownload() {
	atomic.StoreUint64(&s.totalDownload, 0)
}

func (s *Statistic) GetTotalDownload() uint64 {
	return atomic.LoadUint64(&s.totalDownload)
}

func (s *Statistic) SetTotalDownload(b uint64) {
	atomic.StoreUint64(&s.totalDownload, b)
}
