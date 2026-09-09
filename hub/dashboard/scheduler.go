package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler : une instance cron par projet (fuseau du projet), rechargée à chaque modification.
type Scheduler struct {
	s    *Server
	mu   sync.Mutex
	runs map[int64]*cron.Cron // par projet
}

func newScheduler(s *Server) *Scheduler { return &Scheduler{s: s, runs: map[int64]*cron.Cron{}} }

func (sc *Scheduler) ReloadAll() {
	rows, err := sc.s.db.Query(`SELECT id FROM projects`)
	if err != nil {
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close() // ne jamais garder un curseur ouvert pendant d'autres requêtes
	for _, id := range ids {
		sc.Reload(id)
	}
}

// Reload reconstruit les tâches d'un projet.
func (sc *Scheduler) Reload(projectID int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if c, ok := sc.runs[projectID]; ok {
		c.Stop()
		delete(sc.runs, projectID)
	}
	p, err := sc.s.getProjectByID(projectID)
	if err != nil {
		return
	}
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		loc = time.UTC
	}
	c := cron.New(cron.WithLocation(loc))
	autos, _ := sc.s.listAutomations(projectID)
	n := 0
	for _, a := range autos {
		if !a.Enabled {
			continue
		}
		id := a.ID
		if _, err := c.AddFunc(a.Cron, func() { sc.runAutomation(id) }); err != nil {
			log.Printf("[cron] automatisation %d (%s) ignorée : %v", a.ID, a.Cron, err)
			continue
		}
		n++
	}
	c.Start()
	sc.runs[projectID] = c
	if n > 0 {
		log.Printf("[cron] projet %s : %d automatisation(s) actives (%s)", p.Slug, n, loc)
	}
}

// NextRuns retourne la prochaine exécution de chaque automatisation active du projet.
func (sc *Scheduler) NextRun(projectID int64, cronExpr string) string {
	p, err := sc.s.getProjectByID(projectID)
	if err != nil {
		return ""
	}
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		loc = time.UTC
	}
	sched, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return ""
	}
	return sched.Next(time.Now().In(loc)).Format("2006-01-02 15:04")
}

func ValidateCron(expr string) error {
	_, err := cron.ParseStandard(expr)
	return err
}

func (sc *Scheduler) runAutomation(id int64) {
	var actionID int64
	var name string
	if err := sc.s.db.QueryRow(`SELECT action_id, name FROM automations WHERE id=? AND enabled=1`, id).Scan(&actionID, &name); err != nil {
		return
	}
	res := sc.s.RunAction(context.Background(), actionID, &id)
	sc.s.db.Exec(`UPDATE automations SET last_run=?, last_result=? WHERE id=?`, now(), res, id)
	log.Printf("[cron] %s : %s", name, res)
	var pid int64
	sc.s.db.QueryRow(`SELECT project_id FROM automations WHERE id=?`, id).Scan(&pid)
	level := "info"
	if strings.HasPrefix(res, "erreur") {
		level = "error"
	}
	sc.s.logEvent(pid, nil, "automation.run", level, fmt.Sprintf("Automatisation « %s » : %s", name, res))
}

// RunAction exécute une action : côté hub (requête HTTP directe) ou côté agent (commande en file).
// Retourne un résumé texte stocké comme dernier résultat.
func (s *Server) RunAction(ctx context.Context, actionID int64, automationID *int64) string {
	return s.RunActionWith(ctx, actionID, automationID, nil)
}

// RunActionWith exécute une action en fournissant un paramètre d'exécution, substitué à
// {{ value }}. Un curseur du tableau de bord s'en sert pour transmettre sa position.
func (s *Server) RunActionWith(ctx context.Context, actionID int64, automationID *int64, value *float64) string {
	a, err := s.getAction(actionID)
	if err != nil {
		return "action introuvable"
	}
	// Les variables sont résolues avant exécution : une action générique peut viser
	// l'adresse d'un sous-appareil sans être dupliquée pour chaque appareil.
	reqURL, headers, body, rerr := s.resolveAction(a, value)
	if rerr != nil {
		res := "erreur : " + rerr.Error()
		s.db.Exec(`UPDATE actions SET last_run=?, last_result=? WHERE id=?`, now(), res, a.ID)
		return res
	}
	var res string
	switch a.Kind {
	case "agent":
		if a.DeviceID == nil {
			res = "erreur : aucun appareil associé"
			break
		}
		d, err := s.getDevice(*a.DeviceID)
		if err != nil || d.Status != "approved" {
			res = "erreur : appareil non approuvé"
			break
		}
		id, err := s.queueCommand(d.ID, "http_request", map[string]any{
			"method": a.Method, "url": reqURL, "headers": headers, "body": body, "timeout_seconds": a.TimeoutSeconds,
		}, &a.ID, automationID)
		if err != nil {
			res = "erreur : " + err.Error()
			break
		}
		if !d.Online {
			res = fmt.Sprintf("en file (commande %d) : appareil hors ligne, sera exécutée à sa reconnexion", id)
		} else {
			res = fmt.Sprintf("en file (commande %d) : envoyée à %s", id, d.Name)
		}
	default:
		res = doHTTP(ctx, a.Method, reqURL, headers, body, time.Duration(a.TimeoutSeconds)*time.Second)
	}
	s.db.Exec(`UPDATE actions SET last_run=?, last_result=? WHERE id=?`, now(), res, a.ID)
	return res
}

func doHTTP(ctx context.Context, method, url string, headers map[string]string, body string, timeout time.Duration) string {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), url, strings.NewReader(body))
	if err != nil {
		return "erreur : " + err.Error()
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "erreur : " + err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return fmt.Sprintf("%s en %d ms : %s", resp.Status, time.Since(start).Milliseconds(), strings.TrimSpace(string(b)))
}
