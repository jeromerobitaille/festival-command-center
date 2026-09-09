package main

import "testing"

func TestActionNeedsValue(t *testing.T) {
	cases := []struct {
		a    Action
		want bool
	}{
		{Action{URL: "http://x/api", Body: `{"v":{{ value }}}`}, true},
		{Action{URL: "http://x/set?v={{value}}"}, true},
		{Action{URL: "http://x/api", Headers: map[string]string{"X-V": "{{ value }}"}}, true},
		{Action{URL: "http://{{ ecran_01.ip }}/api", Body: `{"v":1}`}, false},
		{Action{URL: "http://x/api"}, false},
		{Action{URL: "http://x/{{ values }}"}, false}, // nom voisin, pas la variable
	}
	for i, c := range cases {
		if got := ActionNeedsValue(&c.a); got != c.want {
			t.Errorf("cas %d : %v, attendu %v (url=%q body=%q)", i, got, c.want, c.a.URL, c.a.Body)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	for in, want := range map[float64]string{60: "60", 0: "0", 7.5: "7.5", -3: "-3", 100: "100"} {
		if got := formatNumber(in); got != want {
			t.Errorf("formatNumber(%v) = %q, attendu %q", in, got, want)
		}
	}
}

// Une action paramétrée doit échouer tant qu'aucune valeur n'est fournie : mieux vaut
// refuser que d'envoyer « {{ value }} » littéral à un équipement.
func TestValeurDExecution(t *testing.T) {
	vars := map[string]string{"e.ip": "10.0.0.1"}

	if _, miss := resolveVars(`{"brightness":{{ value }}}`, vars); len(miss) != 1 || miss[0] != "value" {
		t.Errorf("sans valeur fournie, {{ value }} devait manquer, obtenu %v", miss)
	}

	vars["value"] = formatNumber(65)
	got, miss := resolveVars(`http://{{ e.ip }}/set?b={{ value }}`, vars)
	if len(miss) != 0 {
		t.Errorf("variables manquantes : %v", miss)
	}
	want := "http://10.0.0.1/set?b=65"
	if got != want {
		t.Errorf("URL = %q, attendu %q", got, want)
	}

	vars["value"] = formatNumber(7.5)
	if got, _ := resolveVars(`{"v":{{ value }}}`, vars); got != `{"v":7.5}` {
		t.Errorf("corps = %q", got)
	}

	// Le schéma reste vérifiable après substitution.
	if _, err := parseAndCheck(want); err != nil {
		t.Errorf("URL valide refusee : %v", err)
	}
}
