package handler

import (
	"log/slog"
	"net/http"
	"sort"

	"github.com/kamil-koziol/pingo/internal/db"
	"github.com/kamil-koziol/pingo/internal/web/views"
)

type Dashboard struct {
	q  db.Querier
	db db.DBTX
}

func NewDashboard(q db.Querier, db db.DBTX) *Dashboard {
	return &Dashboard{q: q, db: db}
}

func (h *Dashboard) Index(w http.ResponseWriter, r *http.Request) {
	stats, items, err := h.load(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := views.Dashboard(stats, items).Render(r.Context(), w); err != nil {
		slog.Error("render dashboard", "err", err)
	}
}

// Content renders only the polled elements
func (h *Dashboard) Content(w http.ResponseWriter, r *http.Request) {
	stats, items, err := h.load(r)
	if err != nil {
		h.fail(w, err)
		return
	}
	if err := views.ContentWrapper(stats, items).Render(r.Context(), w); err != nil {
		slog.Error("render content", "err", err)
	}
}

func (h *Dashboard) load(r *http.Request) (views.Stats, []views.ServiceStatus, error) {
	ctx := r.Context()

	services, err := h.q.ListServices(ctx, h.db)
	if err != nil {
		return views.Stats{}, nil, err
	}

	ids := make([]int64, len(services))
	for i, s := range services {
		ids[i] = s.ID
	}

	latest, err := h.q.ListLatestPings(ctx, h.db, ids)
	if err != nil {
		return views.Stats{}, nil, err
	}

	latestByService := map[int64]*db.Ping{}
	for _, p := range latest {
		latestByService[p.Ping.ServiceID] = &p.Ping
	}

	items := make([]views.ServiceStatus, 0, len(services))
	var stats views.Stats

	for _, s := range services {
		it := views.ServiceStatus{Service: s, Ping: latestByService[s.ID]}
		items = append(items, it)

		stats.Total++
		switch {
		case !s.IsActive:
			stats.Paused++
		case it.Ping != nil && it.Ping.IsUp:
			stats.Up++
		case it.Ping != nil:
			stats.Down++
		}
	}

	rank := func(s views.ServiceStatus) int {
		switch {
		case !s.Service.IsActive:
			return 3
		case s.Ping == nil:
			return 1
		case !s.Ping.IsUp:
			return 0
		default:
			return 2
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		ri, rj := rank(items[i]), rank(items[j])
		if ri != rj {
			return ri < rj
		}
		return items[i].Service.Name < items[j].Service.Name
	})

	return stats, items, nil
}

func (h *Dashboard) fail(w http.ResponseWriter, err error) {
	slog.Error("dashboard", "err", err)
	http.Error(w, "something went wrong", http.StatusInternalServerError)
}
