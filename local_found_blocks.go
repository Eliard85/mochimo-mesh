package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/NickP005/go_mcminterface"
)

const (
	foundBlocksHistoryLimit = 200
	foundBlocksPollInterval = 200 * time.Millisecond
)

var FOUND_BLOCKS_HISTORY_PATH = ""

func initLocalFoundBlocks() {
	Globals.FoundBlocksMu.Lock()
	if Globals.FoundBlockHashes == nil {
		Globals.FoundBlockHashes = make(map[string]struct{})
	}
	Globals.FoundBlocksMu.Unlock()

	if err := loadLocalFoundBlocksHistory(); err != nil {
		mlog(3, "§binitLocalFoundBlocks(): §6Unable to load found blocks history: %s", err)
	}

	refreshRecentBlocksCache()

	go monitorLocalFoundBlocks()
}

func monitorLocalFoundBlocks() {
	ticker := time.NewTicker(foundBlocksPollInterval)
	defer ticker.Stop()

	scanLocalFoundBlockFiles()
	for range ticker.C {
		scanLocalFoundBlockFiles()
	}
}

func scanLocalFoundBlockFiles() {
	for _, path := range candidateFoundBlockFiles() {
		summary, err := readLocalFoundBlockFile(path)
		if err != nil {
			continue
		}
		if addLocalFoundBlock(summary) {
			mlog(2, "§bscanLocalFoundBlockFiles(): §2Recorded local found block §e%d §f%s", summary.Number, summary.Hash)
		}
	}
}

func candidateFoundBlockFiles() []string {
	workdir := filepath.Dir(TFILE_PATH)
	return []string{
		filepath.Join(workdir, "mblock.dat"),
		filepath.Join(workdir, "cblock.dat"),
	}
}

func readLocalFoundBlockFile(path string) (RecentBlockSummary, error) {
	blockBytes, err := os.ReadFile(path)
	if err != nil {
		return RecentBlockSummary{}, err
	}
	if len(blockBytes) < 160 {
		return RecentBlockSummary{}, fmt.Errorf("block file too short")
	}

	block := go_mcminterface.BlockFromBytes(blockBytes)
	if block.Header.Hdrlen == 0 || block.Trailer.Bhash == ([32]byte{}) {
		return RecentBlockSummary{}, fmt.Errorf("unsolved block")
	}

	return summarizeRecentBlock(block), nil
}

func addLocalFoundBlock(summary RecentBlockSummary) bool {
	if summary.Hash == "" {
		return false
	}

	Globals.FoundBlocksMu.Lock()
	defer Globals.FoundBlocksMu.Unlock()

	if Globals.FoundBlockHashes == nil {
		Globals.FoundBlockHashes = make(map[string]struct{})
	}
	if _, exists := Globals.FoundBlockHashes[summary.Hash]; exists {
		return false
	}

	Globals.FoundBlocksHistory = append([]RecentBlockSummary{summary}, Globals.FoundBlocksHistory...)
	Globals.FoundBlockHashes[summary.Hash] = struct{}{}
	if len(Globals.FoundBlocksHistory) > foundBlocksHistoryLimit {
		trimmed := Globals.FoundBlocksHistory[foundBlocksHistoryLimit:]
		for _, block := range trimmed {
			delete(Globals.FoundBlockHashes, block.Hash)
		}
		Globals.FoundBlocksHistory = Globals.FoundBlocksHistory[:foundBlocksHistoryLimit]
	}

	if err := saveLocalFoundBlocksHistoryLocked(); err != nil {
		mlog(3, "§baddLocalFoundBlock(): §6Unable to save found blocks history: %s", err)
	}

	go refreshRecentBlocksCache()
	return true
}

func getLocalFoundBlocksHistorySnapshot() []RecentBlockSummary {
	Globals.FoundBlocksMu.RLock()
	defer Globals.FoundBlocksMu.RUnlock()

	if len(Globals.FoundBlocksHistory) == 0 {
		return nil
	}

	return append([]RecentBlockSummary(nil), Globals.FoundBlocksHistory...)
}

func loadLocalFoundBlocksHistory() error {
	data, err := os.ReadFile(foundBlocksHistoryPath())
	if err != nil {
		if os.IsNotExist(err) {
			data, err = os.ReadFile(legacyFoundBlocksHistoryPath())
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
		} else {
			return err
		}
	}

	var blocks []RecentBlockSummary
	if err := json.Unmarshal(data, &blocks); err != nil {
		return err
	}

	Globals.FoundBlocksMu.Lock()
	defer Globals.FoundBlocksMu.Unlock()

	Globals.FoundBlocksHistory = nil
	Globals.FoundBlockHashes = make(map[string]struct{}, len(blocks))
	for _, block := range blocks {
		if block.Hash == "" {
			continue
		}
		if _, exists := Globals.FoundBlockHashes[block.Hash]; exists {
			continue
		}
		Globals.FoundBlocksHistory = append(Globals.FoundBlocksHistory, block)
		Globals.FoundBlockHashes[block.Hash] = struct{}{}
		if len(Globals.FoundBlocksHistory) >= foundBlocksHistoryLimit {
			break
		}
	}

	return nil
}

func saveLocalFoundBlocksHistoryLocked() error {
	if err := os.MkdirAll(filepath.Dir(foundBlocksHistoryPath()), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(Globals.FoundBlocksHistory, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(foundBlocksHistoryPath(), data, 0o644)
}

func foundBlocksHistoryPath() string {
	return FOUND_BLOCKS_HISTORY_PATH
}

func legacyFoundBlocksHistoryPath() string {
	return filepath.Join("data", "found-blocks.json")
}
