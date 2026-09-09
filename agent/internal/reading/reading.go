// Package reading interroge un sous-appareil pour en extraire une valeur affichable.
//
// Chaque sonde est une requête HTTP vers l'équipement, dont on extrait une valeur par
// chemin JSON pointé ou par expression régulière. Le résultat est une chaîne : le portail
// l'affiche telle quelle et n'a pas à connaître le type.
package reading

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Conf décrit une sonde, telle que poussée par le portail dans la configuration.
type Conf struct {
	Name   string `json:"name"`             // « nits », « température »…
	Method string `json:"method,omitempty"` // GET par défaut
	Path   string `json:"path"`             // chemin sur l'équipement, ex. /api/brightness
	Body   string `json:"body,omitempty"`
	JSON   string `json:"json,omitempty"`  // chemin pointé, ex. « data.brightness » ou « items.0.value »
	Regex  string `json:"regex,omitempty"` // à défaut de JSON : premier groupe capturant
	Unit   string `json:"unit,omitempty"`
}

// Result est ce qui remonte au portail.
type Result struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
	Unit  string `json:"unit,omitempty"`
	Error string `json:"error,omitempty"`
}

var client = &http.Client{Timeout: 4 * time.Second}

// Read exécute une sonde contre l'équipement à l'adresse ip:port.
func Read(ctx context.Context, ip string, port int, c Conf) Result {
	res := Result{Name: c.Name, Unit: c.Unit}
	method := strings.ToUpper(strings.TrimSpace(c.Method))
	if method == "" {
		method = http.MethodGet
	}
	path := c.Path
	if path == "" {
		path = "/"
	} else if path[0] != '/' {
		path = "/" + path
	}
	url := "http://" + net.JoinHostPort(ip, strconv.Itoa(port)) + path

	cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, method, url, strings.NewReader(c.Body))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if c.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		res.Error = shortErr(err)
		return res
	}
	defer resp.Body.Close()
	// Corps borné : un équipement bavard ne doit pas remplir la mémoire de l'agent.
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		res.Error = shortErr(err)
		return res
	}
	if resp.StatusCode/100 != 2 {
		res.Error = "HTTP " + resp.Status
		return res
	}
	v, err := extract(b, c)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Value = v
	return res
}

// extract applique le chemin JSON, sinon l'expression régulière, sinon renvoie le corps.
func extract(b []byte, c Conf) (string, error) {
	if p := strings.TrimSpace(c.JSON); p != "" {
		var doc any
		if err := json.Unmarshal(b, &doc); err != nil {
			return "", fmt.Errorf("réponse non JSON")
		}
		v, err := walk(doc, p)
		if err != nil {
			return "", err
		}
		return stringify(v), nil
	}
	if r := strings.TrimSpace(c.Regex); r != "" {
		re, err := regexp.Compile(r)
		if err != nil {
			return "", fmt.Errorf("expression régulière invalide : %w", err)
		}
		m := re.FindSubmatch(b)
		if m == nil {
			return "", fmt.Errorf("aucune correspondance")
		}
		if len(m) > 1 {
			return string(m[1]), nil
		}
		return string(m[0]), nil
	}
	s := strings.TrimSpace(string(b))
	if len(s) > 120 {
		s = s[:120]
	}
	return s, nil
}

// walk suit un chemin pointé. Un segment numérique indexe un tableau.
func walk(doc any, path string) (any, error) {
	cur := doc
	for _, seg := range strings.Split(path, ".") {
		if seg == "" {
			continue
		}
		switch v := cur.(type) {
		case map[string]any:
			x, ok := v[seg]
			if !ok {
				return nil, fmt.Errorf("champ %q absent", seg)
			}
			cur = x
		case []any:
			i, err := strconv.Atoi(seg)
			if err != nil || i < 0 || i >= len(v) {
				return nil, fmt.Errorf("indice %q hors du tableau", seg)
			}
			cur = v[i]
		default:
			return nil, fmt.Errorf("champ %q : la valeur parente n'est pas un objet", seg)
		}
	}
	return cur, nil
}

func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "oui"
		}
		return "non"
	case float64:
		// Les entiers ne doivent pas s'afficher « 1200.000000 ».
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	b, _ := json.Marshal(v)
	s := string(b)
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

// shortErr raccourcit les erreurs réseau, très verbeuses, pour l'affichage dans le portail.
func shortErr(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i > 0 && len(s)-i < 60 {
		return s[i+2:]
	}
	if len(s) > 80 {
		return s[:80]
	}
	return s
}
