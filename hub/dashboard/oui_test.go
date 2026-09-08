package main

import "testing"

func TestVendorOf(t *testing.T) {
	// Préfixes tirés du registre embarqué : on vérifie que la table se charge et
	// que la normalisation accepte les deux séparateurs.
	cases := map[string]bool{
		"34:60:f9:fd:84:4c": true, // OUI attribué
		"c4:38:75:9a:fa:64": true,
		"C4-38-75-9A-FA-64": true,  // séparateur Windows, majuscules
		"da:02:f4:a3:21:ae": false, // localement administrée
		"":                  false,
		"zz:zz":             false,
	}
	for mac, wantFound := range cases {
		got := VendorOf(mac)
		if wantFound && got == "" {
			t.Errorf("VendorOf(%q) = \"\", un fabricant était attendu", mac)
		}
		if !wantFound && got != "" {
			t.Errorf("VendorOf(%q) = %q, rien n'était attendu", mac, got)
		}
	}
	if a, b := VendorOf("c4:38:75:9a:fa:64"), VendorOf("C4-38-75-9A-FA-64"); a != b {
		t.Errorf("séparateurs différents donnent %q et %q", a, b)
	}
}

func TestMACIsRandom(t *testing.T) {
	for mac, want := range map[string]bool{
		"34:60:f9:fd:84:4c": false, // 0x34 : bit local à 0
		"da:02:f4:a3:21:ae": true,  // 0xda : bit local à 1
		"6a:12:bd:1b:90:82": true,
		"c0:bf:be:ef:64:36": false,
	} {
		if got := MACIsRandom(mac); got != want {
			t.Errorf("MACIsRandom(%q) = %v, attendu %v", mac, got, want)
		}
	}
}
