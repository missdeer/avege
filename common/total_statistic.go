package common

import (
	"sync/atomic"
)

type TotalStatistic struct {
	totalUpload   uint64
	totalDownload uint64
}

func (ts *TotalStatistic) AddUpload(b uint64) {
	atomic.AddUint64(&ts.totalUpload, b)
}

func (ts *TotalStatistic) GetUpload() uint64 {
	return atomic.LoadUint64(&ts.totalUpload)
}

func (ts *TotalStatistic) SetUpload(b uint64) {
	atomic.StoreUint64(&ts.totalUpload, b)
}

func (ts *TotalStatistic) ResetUpload() uint64 {
	return atomic.SwapUint64(&ts.totalUpload, 0)
}

func (ts *TotalStatistic) AddDownload(b uint64) {
	atomic.AddUint64(&ts.totalDownload, b)
}

func (ts *TotalStatistic) GetDownload() uint64 {
	return atomic.LoadUint64(&ts.totalDownload)
}

func (ts *TotalStatistic) SetDownload(b uint64) {
	atomic.StoreUint64(&ts.totalDownload, b)
}

func (ts *TotalStatistic) ResetDownload() uint64 {
	return atomic.SwapUint64(&ts.totalDownload, 0)
}

var (
	TotalStat = new(TotalStatistic)
	DeltaStat = new(TotalStatistic)
)
