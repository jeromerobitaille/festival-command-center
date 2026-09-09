package reading

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// serveur factice jouant le rôle d'un équipement.
func fake(t *testing.T, h http.HandlerFunc) (string, int) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	u := strings.TrimPrefix(srv.URL, "http://")
	host, p, _ := net.SplitHostPort(u)
	port, _ := strconv.Atoi(p)
	return host, port
}

func TestReadJSON(t *testing.T) {
	ip, port := fake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/brightness" {
			w.WriteHeader(404)
			return
		}
		w.Write([]byte(`{"data":{"brightness":1200,"auto":true},"items":[{"v":7.5}]}`))
	})
	cases := []struct{ path, want string }{
		{"data.brightness", "1200"}, // entier, pas « 1200.000000 »
		{"data.auto", "oui"},
		{"items.0.v", "7.5"},
	}
	for _, c := range cases {
		r := Read(context.Background(), ip, port, Conf{Name: "nits", Path: "/api/brightness", JSON: c.path})
		if r.Error != "" {
			t.Errorf("%s : erreur %q", c.path, r.Error)
		} else if r.Value != c.want {
			t.Errorf("%s = %q, attendu %q", c.path, r.Value, c.want)
		}
	}
}

func TestReadErreurs(t *testing.T) {
	ip, port := fake(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/500":
			w.WriteHeader(500)
		case "/texte":
			w.Write([]byte("brightness=880 nits"))
		default:
			w.Write([]byte(`{"a":1}`))
		}
	})
	if r := Read(context.Background(), ip, port, Conf{Path: "/500"}); r.Error == "" {
		t.Error("un 500 aurait du produire une erreur")
	}
	if r := Read(context.Background(), ip, port, Conf{Path: "/", JSON: "absent"}); r.Error == "" {
		t.Error("un champ absent aurait du produire une erreur")
	}
	// Extraction par expression régulière sur une réponse non JSON.
	r := Read(context.Background(), ip, port, Conf{Path: "/texte", Regex: `brightness=(\d+)`})
	if r.Value != "880" {
		t.Errorf("regex : %q / erreur %q", r.Value, r.Error)
	}
	// Sans JSON ni regex : le corps entier.
	if r := Read(context.Background(), ip, port, Conf{Path: "/texte"}); r.Value != "brightness=880 nits" {
		t.Errorf("corps brut : %q", r.Value)
	}
	// Équipement injoignable : erreur courte, pas de blocage.
	if r := Read(context.Background(), "127.0.0.1", 1, Conf{Path: "/"}); r.Error == "" {
		t.Error("un port ferme aurait du produire une erreur")
	} else if len(r.Error) > 80 {
		t.Errorf("erreur trop verbeuse : %q", r.Error)
	}
}
