package dc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

var publicConfigRe = regexp.MustCompile(`^\s*proxy_for\s+(\d+)\s+(\S+?)?;\s*$`)

type PublicConfigUpdater struct {
	updater

	http *http.Client
	tg   *Telegram
}

func (p PublicConfigUpdater) Run(ctx context.Context, url, network string) {
	p.run(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			panic(err)
		}

		resp, err := p.http.Do(req)
		if err != nil {
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
			return fmt.Errorf("cannot fetch url %s: %w", url, err)
		}

		if resp.StatusCode >= http.StatusBadRequest {
			return fmt.Errorf("unexpected status code from %s: %d", url, resp.StatusCode)
		}

		scanner := bufio.NewScanner(resp.Body)
		addrs := map[int][]Addr{}

		for scanner.Scan() {
			matches := publicConfigRe.FindStringSubmatch(scanner.Text())
			if len(matches) != 3 {
				continue
			}

			dc, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}

			switch dc {
			// this is a list of DC we currently support. Other are ignored.
			case 203: // CDN DC
				p.logger.Info(fmt.Sprintf("found %s address for DC %d", matches[2], dc))
				addrs[dc] = append(addrs[dc], Addr{
					Network: network,
					Address: matches[2],
				})
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("cannot read response body from %s: %w", url, err)
		}

		p.tg.lock.Lock()
		defer p.tg.lock.Unlock()

		if network == "tcp4" {
			p.tg.view.publicConfigs.v4 = addrs
		} else {
			p.tg.view.publicConfigs.v6 = addrs
		}

		return nil
	})
}

func NewPublicConfigUpdater(tg *Telegram, logger Logger, client *http.Client) PublicConfigUpdater {
	return PublicConfigUpdater{
		updater: updater{
			logger: logger,
			period: PublicConfigUpdateEach,
		},
		http: client,
		tg:   tg,
	}
}
