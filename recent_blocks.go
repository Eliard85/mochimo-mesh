package main

import (
	"encoding/binary"
	"encoding/hex"
	"time"

	"mochimo-mesh/indexer"

	"github.com/NickP005/go_mcminterface"
)

const recentBlocksLimit = 50

type RecentBlockSummary struct {
	Number             uint64 `json:"number"`
	Hash               string `json:"hash"`
	MinerAddress       string `json:"miner_address"`
	MinerAddressBase58 string `json:"miner_address_base58,omitempty"`
	TimestampUnixMilli uint64 `json:"timestamp_unix_milli"`
	TimestampLabel     string `json:"timestamp_label"`
	Age                string `json:"age"`
}

func refreshRecentBlocksCache() {
	blocks := getLocalFoundBlocksHistorySnapshot()
	if len(blocks) == 0 {
		setRecentBlocksSnapshot(nil)
		return
	}

	if len(blocks) > recentBlocksLimit {
		blocks = blocks[:recentBlocksLimit]
	}

	setRecentBlocksSnapshot(blocks)
}

func summarizeRecentBlock(block go_mcminterface.Block) RecentBlockSummary {
	minerAddress := "0x" + hex.EncodeToString(block.Header.Maddr[:])
	minerAddressBase58, err := indexer.AddrTagToBase58(block.Header.Maddr[:])
	if err != nil {
		minerAddressBase58 = ""
	}

	timestamp := uint64(binary.LittleEndian.Uint32(block.Trailer.Stime[:])) * 1000

	return RecentBlockSummary{
		Number:             binary.LittleEndian.Uint64(block.Trailer.Bnum[:]),
		Hash:               "0x" + hex.EncodeToString(block.Trailer.Bhash[:]),
		MinerAddress:       minerAddress,
		MinerAddressBase58: minerAddressBase58,
		TimestampUnixMilli: timestamp,
		TimestampLabel:     formatBlockTimestamp(timestamp),
		Age:                formatBlockAge(timestamp),
	}
}

func setRecentBlocksSnapshot(blocks []RecentBlockSummary) {
	Globals.RecentBlocksMu.Lock()
	defer Globals.RecentBlocksMu.Unlock()

	if len(blocks) == 0 {
		Globals.RecentBlocks = nil
		return
	}

	Globals.RecentBlocks = append([]RecentBlockSummary(nil), blocks...)
}

func getRecentBlocksSnapshot() []RecentBlockSummary {
	Globals.RecentBlocksMu.RLock()
	defer Globals.RecentBlocksMu.RUnlock()

	if len(Globals.RecentBlocks) == 0 {
		return nil
	}

	snapshot := append([]RecentBlockSummary(nil), Globals.RecentBlocks...)
	for i := range snapshot {
		snapshot[i].TimestampLabel = formatBlockTimestamp(snapshot[i].TimestampUnixMilli)
		snapshot[i].Age = formatBlockAge(snapshot[i].TimestampUnixMilli)
	}

	return snapshot
}

func formatBlockTimestamp(ts uint64) string {
	if ts == 0 {
		return "unknown"
	}

	return formatDashboardTime(time.UnixMilli(int64(ts)))
}
