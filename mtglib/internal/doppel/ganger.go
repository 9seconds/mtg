package doppel

import (
	"context"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
)

const (
	DoppelGangerMaxDurations  = 4096
	DoppelGangerScoutRaidEach = 6 * time.Hour
	DoppelGangerScoutRepeats  = 10
)

type gangerConnRequest struct {
	ret     chan<- Conn
	payload essentials.Conn
}

type Ganger struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	logger    Logger
	wg        sync.WaitGroup

	scout            Scout
	scoutRaidEach    time.Duration
	scoutRaidRepeats int

	stats     *Stats
	durations []time.Duration

	connRequests chan gangerConnRequest
}

func (g *Ganger) Shutdown() {
	g.ctxCancel()
	g.wg.Wait()
}

func (g *Ganger) Run() {
	g.wg.Go(func() {
		g.run()
	})
}

func (g *Ganger) NewConn(conn essentials.Conn) (Conn, error) {
	rvChan := make(chan Conn)
	req := gangerConnRequest{
		ret:     rvChan,
		payload: conn,
	}
	defer close(req.ret)

	select {
	case <-g.ctx.Done():
		return Conn{}, context.Cause(g.ctx)
	case g.connRequests <- req:
	}

	select {
	case <-g.ctx.Done():
		return Conn{}, context.Cause(g.ctx)
	case conn := <-rvChan:
		return conn, nil
	}
}

func (g *Ganger) run() {
	scoutTicker := time.NewTicker(g.scoutRaidEach)
	defer func() {
		scoutTicker.Stop()

		select {
		case <-scoutTicker.C:
		default:
		}
	}()

	scoutCollectedChan := make(chan []time.Duration)
	currentScoutCollectedChan := scoutCollectedChan

	updatedStatsChan := make(chan *Stats)

	g.wg.Go(func() {
		g.runScoutRaid(scoutCollectedChan)
	})

	for {
		select {
		case <-g.ctx.Done():
			return
		case durations := <-currentScoutCollectedChan:
			g.durations = append(g.durations, durations...)

			if len(g.durations) > DoppelGangerMaxDurations {
				g.durations = g.durations[len(g.durations)-DoppelGangerMaxDurations:]
			}

			if len(g.durations) < MinDurationsToCalculate {
				continue
			}

			currentScoutCollectedChan = nil
			g.wg.Go(func() {
				select {
				case <-g.ctx.Done():
				case updatedStatsChan <- NewStats(durations):
				}
			})
		case stats := <-updatedStatsChan:
			g.stats = stats
			currentScoutCollectedChan = scoutCollectedChan
		case <-scoutTicker.C:
			g.wg.Go(func() {
				g.runScoutRaid(scoutCollectedChan)
			})
		case req := <-g.connRequests:
			select {
			case <-g.ctx.Done():
			case req.ret <- NewConn(g.ctx, req.payload, g.stats):
			}
		}
	}
}

func (g *Ganger) runScoutRaid(rvChan chan<- []time.Duration) {
	durations := []time.Duration{}

	for range g.scoutRaidRepeats {
		learned, err := g.scout.Learn(g.ctx)
		if err != nil {
			g.logger.WarningError("cannot learn", err)
			continue
		}
		durations = append(durations, learned...)
	}

	select {
	case <-g.ctx.Done():
		return
	case rvChan <- durations:
	}
}

func NewGanger(
	ctx context.Context,
	network Network,
	logger Logger,
	scoutEach time.Duration,
	scoutRepeats int,
	urls []string,
) *Ganger {
	ctx, cancel := context.WithCancel(ctx)

	if scoutEach == 0 {
		scoutEach = DoppelGangerScoutRaidEach
	}

	if scoutRepeats == 0 {
		scoutRepeats = DoppelGangerScoutRepeats
	}

	return &Ganger{
		ctx:              ctx,
		ctxCancel:        cancel,
		logger:           logger,
		scoutRaidEach:    scoutEach,
		scoutRaidRepeats: scoutRepeats,
		stats: &Stats{
			k:      StatsDefaultK,
			lambda: StatsDefaultLambda,
		},
		scout:        NewScout(network, urls),
		connRequests: make(chan gangerConnRequest),
	}
}
