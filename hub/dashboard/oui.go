package main

// Résolution du fabricant à partir de l'adresse MAC, via le registre OUI de l'IEEE.
//
// La table vit ici plutôt que dans l'agent : le binaire de l'agent part en OTA sur des
// liaisons de terrain, et cette résolution n'a pas besoin d'être faite au plus près.
// Régénérer data/oui.tsv.gz avec scripts/build-oui.sh.

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"strings"
	"sync"
)

//go:embed data/oui.tsv.gz
var ouiGz []byte

var (
	ouiOnce sync.Once
	ouiMap  map[string]string
)

func loadOUI() {
	ouiMap = map[string]string{}
	zr, err := gzip.NewReader(bytes.NewReader(ouiGz))
	if err != nil {
		return
	}
	defer zr.Close()
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "\t")
		if ok {
			ouiMap[k] = v
		}
	}
}

// MACIsRandom signale une adresse localement administrée : les téléphones et portables
// récents en génèrent une par réseau. Aucun fabricant n'est déductible dans ce cas.
func MACIsRandom(mac string) bool {
	if len(mac) < 2 {
		return false
	}
	b := hexVal(mac[0])<<4 | hexVal(mac[1])
	return b >= 0 && b&0x02 != 0
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// VendorOf renvoie le fabricant déclaré pour le préfixe de l'adresse, ou "" si inconnu.
func VendorOf(mac string) string {
	if len(mac) < 8 || MACIsRandom(mac) {
		return ""
	}
	ouiOnce.Do(loadOUI)
	key := strings.ToLower(strings.NewReplacer(":", "", "-", "").Replace(mac))
	if len(key) < 6 {
		return ""
	}
	return ouiMap[key[:6]]
}
