package doppel

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
)

const (
	DoppelGangerMaxDurations  = 4096
	DoppelGangerScoutRaidEach = 6 * time.Hour
	DoppelGangerScoutRepeats  = 10

	MinCertSizesToCalculate = 3
)

// NoiseParams holds the measured cert chain size for FakeTLS noise calibration.
// If Mean is 0, the caller should use a legacy fallback.
type NoiseParams struct {
	Mean   int
	Jitter int
}

type scoutRaidResult struct {
	durations []time.Duration
	certSizes []int
}

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

	drs bool

	stats     *Stats
	durations []time.Duration
	certSizes []int

	noiseParams atomic.Pointer[NoiseParams]

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

// NoiseParams returns the current cert-size-based noise parameters.
// Returns zero-value NoiseParams if not yet measured (caller should use fallback).
func (g *Ganger) NoiseParams() NoiseParams {
	if p := g.noiseParams.Load(); p != nil {
		return *p
	}

	return NoiseParams{}
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

	scoutCollectedChan := make(chan scoutRaidResult)
	currentScoutCollectedChan := scoutCollectedChan

	updatedStatsChan := make(chan *Stats)

	g.wg.Go(func() {
		g.runScoutRaid(scoutCollectedChan)
	})

	for {
		select {
		case <-g.ctx.Done():
			return
		case result := <-currentScoutCollectedChan:
			g.durations = append(g.durations, result.durations...)

			if len(g.durations) > DoppelGangerMaxDurations {
				copy(g.durations, g.durations[len(g.durations)-DoppelGangerMaxDurations:])
				g.durations = g.durations[:DoppelGangerMaxDurations]
			}

			// Update cert sizes and recompute noise params.
			g.certSizes = append(g.certSizes, result.certSizes...)
			if len(g.certSizes) > DoppelGangerMaxDurations {
				g.certSizes = g.certSizes[len(g.certSizes)-DoppelGangerMaxDurations:]
			}

			if len(g.certSizes) >= MinCertSizesToCalculate {
				g.updateNoiseParams()
			}

			if len(g.durations) < MinDurationsToCalculate {
				continue
			}

			durations := g.durations
			currentScoutCollectedChan = nil
			g.wg.Go(func() {
				select {
				case <-g.ctx.Done():
				case updatedStatsChan <- NewStats(durations, g.drs):
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

func (g *Ganger) updateNoiseParams() {
	if len(g.certSizes) == 0 {
		return
	}

	sum := 0
	for _, s := range g.certSizes {
		sum += s
	}

	mean := sum / len(g.certSizes)

	maxDev := 0
	for _, s := range g.certSizes {
		d := s - mean
		if d < 0 {
			d = -d
		}

		if d > maxDev {
			maxDev = d
		}
	}

	if maxDev < 100 {
		maxDev = 100
	}

	np := &NoiseParams{Mean: mean, Jitter: maxDev}
	g.noiseParams.Store(np)

	g.logger.Info(fmt.Sprintf(
		"updated noise params: mean=%d jitter=%d samples=%d",
		mean, maxDev, len(g.certSizes),
	))
}

func (g *Ganger) runScoutRaid(rvChan chan<- scoutRaidResult) {
	var result scoutRaidResult

	for range g.scoutRaidRepeats {
		learned, err := g.scout.Learn(g.ctx)
		if err != nil {
			g.logger.WarningError("cannot learn", err)
			continue
		}

		result.durations = append(result.durations, learned.Durations...)

		if learned.CertSize > 0 {
			result.certSizes = append(result.certSizes, learned.CertSize)
		}
	}

	select {
	case <-g.ctx.Done():
		return
	case rvChan <- result:
	}
}

func NewGanger(
	ctx context.Context,
	network Network,
	logger Logger,
	scoutEach time.Duration,
	scoutRepeats int,
	urls []string,
	drs bool,
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
		drs:              drs,
		stats: &Stats{
			k:      StatsDefaultK,
			lambda: StatsDefaultLambda,
			drs:    drs,
		},
		scout:        NewScout(network, urls),
		connRequests: make(chan gangerConnRequest),
	}
}
