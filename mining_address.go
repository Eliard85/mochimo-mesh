package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/NickP005/go_mcminterface"
	"github.com/btcsuite/btcutil/base58"
	"github.com/sigurn/crc16"
)

func initLocalMinerTag() {
	tag, source, err := discoverLocalMinerTag()
	if err != nil {
		Globals.HasLocalMinerTag = false
		mlog(3, "§binitLocalMinerTag(): §6Local mining address not detected: %s", err)
		return
	}

	copy(Globals.LocalMinerTag[:], tag)
	Globals.HasLocalMinerTag = true
	mlog(2, "§binitLocalMinerTag(): §2Using local mining address from §f%s", source)
}

func discoverLocalMinerTag() ([]byte, string, error) {
	if Globals.MiningAddress != "" {
		tag, err := parseMiningAddress(Globals.MiningAddress)
		if err != nil {
			return nil, "", fmt.Errorf("invalid -maddr value: %w", err)
		}
		return tag, "flag/env maddr", nil
	}

	if Globals.MiningAddressFile != "" {
		tag, err := readMiningAddressFile(Globals.MiningAddressFile)
		if err != nil {
			return nil, "", fmt.Errorf("invalid -maddr-file value: %w", err)
		}
		return tag, Globals.MiningAddressFile, nil
	}

	for _, candidate := range autoMiningAddressFiles() {
		tag, err := readMiningAddressFile(candidate)
		if err == nil {
			return tag, candidate, nil
		}
	}

	tag, source, err := discoverMinerTagFromProcesses()
	if err == nil {
		return tag, source, nil
	}

	return nil, "", err
}

func autoMiningAddressFiles() []string {
	candidates := []string{
		filepath.Join(filepath.Dir(TFILE_PATH), "maddr.dat"),
		filepath.Join(filepath.Dir(TFILE_PATH), "..", "maddr.dat"),
		filepath.Join(filepath.Dir(TFILE_PATH), "..", "..", "maddr.dat"),
		"maddr.dat",
	}

	unique := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		unique = append(unique, clean)
	}
	return unique
}

func readMiningAddressFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed != "" {
		if tag, err := parseMiningAddress(trimmed); err == nil {
			return tag, nil
		}
	}

	if len(data) == 20 {
		return append([]byte(nil), data...), nil
	}
	if len(data) == 22 {
		return decodeMiningAddressBytes(data)
	}
	if len(data) == go_mcminterface.ADDR_LEN {
		return append([]byte(nil), data[:go_mcminterface.ADDR_TAG_LEN]...), nil
	}
	if len(data) == go_mcminterface.WOTS_ADDR_LEN {
		addr := go_mcminterface.AddrFromWots(data)
		if len(addr) != go_mcminterface.ADDR_LEN {
			return nil, fmt.Errorf("failed to derive tag from WOTS address")
		}
		return append([]byte(nil), addr[:go_mcminterface.ADDR_TAG_LEN]...), nil
	}

	return nil, fmt.Errorf("unsupported mining address file format")
}

func discoverMinerTagFromProcesses() ([]byte, string, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}

		cmdlinePath := filepath.Join("/proc", entry.Name(), "cmdline")
		raw, err := os.ReadFile(cmdlinePath)
		if err != nil || len(raw) == 0 {
			continue
		}

		args := compactArgs(strings.Split(string(raw), "\x00"))
		if len(args) == 0 {
			continue
		}
		if !isMinerProcess(args[0]) {
			continue
		}

		cwd, _ := os.Readlink(filepath.Join("/proc", entry.Name(), "cwd"))
		tag, source, ok := parseMinerTagFromArgs(args, cwd)
		if ok {
			return tag, source, nil
		}
	}

	return nil, "", fmt.Errorf("no running miner process with -m/--maddr found")
}

func compactArgs(args []string) []string {
	compact := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" {
			continue
		}
		compact = append(compact, arg)
	}
	return compact
}

func isMinerProcess(arg0 string) bool {
	base := filepath.Base(arg0)
	return base == "mochimo" || base == "gpuminer" || base == "miner"
}

func parseMinerTagFromArgs(args []string, cwd string) ([]byte, string, bool) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-m" || arg == "--maddr":
			if i+1 >= len(args) {
				return nil, "", false
			}
			value := args[i+1]
			tag, source, ok := parseMinerTagValue(value, cwd)
			if ok {
				return tag, source, true
			}
		case strings.HasPrefix(arg, "--maddr="):
			value := strings.TrimPrefix(arg, "--maddr=")
			tag, source, ok := parseMinerTagValue(value, cwd)
			if ok {
				return tag, source, true
			}
		}
	}

	return nil, "", false
}

func parseMinerTagValue(value string, cwd string) ([]byte, string, bool) {
	if tag, err := parseMiningAddress(value); err == nil {
		return tag, "running process argument", true
	}

	if tag, err := readMiningAddressFile(resolveProcessPath(value, cwd)); err == nil {
		return tag, resolveProcessPath(value, cwd), true
	}
	if tag, err := readMiningAddressFile(value); err == nil {
		return tag, value, true
	}

	return nil, "", false
}

func resolveProcessPath(path string, cwd string) string {
	if path == "" || cwd == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(cwd, path)
}

func parseMiningAddress(value string) ([]byte, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("empty address")
	}

	if strings.HasPrefix(trimmed, "0x") || strings.HasPrefix(trimmed, "0X") {
		decoded, err := hex.DecodeString(trimmed[2:])
		if err != nil {
			return nil, err
		}
		if len(decoded) != 20 {
			return nil, fmt.Errorf("expected 20-byte hex tag")
		}
		return decoded, nil
	}

	decoded := base58.Decode(trimmed)
	if len(decoded) != 22 {
		return nil, fmt.Errorf("expected 22-byte base58 address")
	}
	return decodeMiningAddressBytes(decoded)
}

func decodeMiningAddressBytes(decoded []byte) ([]byte, error) {
	if len(decoded) != 22 {
		return nil, fmt.Errorf("expected 22 bytes")
	}

	tag := append([]byte(nil), decoded[:20]...)
	table := crc16.MakeTable(crc16.CRC16_XMODEM)
	crc := crc16.Checksum(tag, table)
	if decoded[20] != byte(crc&0xFF) || decoded[21] != byte((crc>>8)&0xFF) {
		return nil, fmt.Errorf("crc mismatch")
	}

	return tag, nil
}
