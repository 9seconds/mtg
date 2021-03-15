package ipblocklist

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/kentik/patricia"
	"github.com/kentik/patricia/bool_tree"
	"github.com/panjf2000/ants"
)

const (
	fireholIPv4DefaultCIDR = 32
	fireholIPv6DefaultCIDR = 128
)

var fireholRegexpComment = regexp.MustCompile(`\s*#.*?$`)

type Firehol struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	logger    mtglib.Logger

	rwMutex sync.RWMutex

	remoteURLs []string
	localFiles []string

	httpClient *http.Client
	workerPool *ants.Pool

	treeV4 *bool_tree.TreeV4
	treeV6 *bool_tree.TreeV6
}

func (f *Firehol) Contains(ip net.IP) bool {
	if ip == nil {
		return true
	}

	ip4 := ip.To4()

	f.rwMutex.RLock()
	defer f.rwMutex.RUnlock()

	if ip4 != nil {
		return f.containsIPv4(ip4)
	}

	return f.containsIPv6(ip.To16())
}

func (f *Firehol) containsIPv4(addr net.IP) bool {
	ip := patricia.NewIPv4AddressFromBytes(addr, 32)

	if ok, _, err := f.treeV4.FindDeepestTag(ip); ok && err == nil {
		return true
	}

	return false
}

func (f *Firehol) containsIPv6(addr net.IP) bool {
	ip := patricia.NewIPv6Address(addr, 128)

	if ok, _, err := f.treeV6.FindDeepestTag(ip); ok && err == nil {
		return true
	}

	return false
}

func (f *Firehol) Run(updateEach time.Duration) {
	ticker := time.NewTicker(updateEach)

	defer func() {
		ticker.Stop()

		select {
		case <-ticker.C:
		default:
		}
	}()

	if err := f.update(); err != nil {
		f.logger.WarningError("cannot update blocklist", err)
	}

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			if err := f.update(); err != nil {
				f.logger.WarningError("cannot update blocklist", err)
			}
		}
	}
}

func (f *Firehol) Shutdown() {
	f.ctxCancel()
}

func (f *Firehol) update() error { // nolint: funlen, cyclop
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(len(f.remoteURLs) + len(f.localFiles))

	treeMutex := &sync.Mutex{}
	v4tree := bool_tree.NewTreeV4()
	v6tree := bool_tree.NewTreeV6()

	errorChan := make(chan error, 1)
	defer close(errorChan)

	for _, v := range f.localFiles {
		go func(filename string) {
			defer wg.Done()

			if err := f.updateLocalFile(ctx, filename, treeMutex, v4tree, v6tree); err != nil {
				cancel()
				f.logger.BindStr("filename", filename).WarningError("cannot update", err)

				select {
				case errorChan <- err:
				default:
				}
			}
		}(v)
	}

	for _, v := range f.remoteURLs {
		value := v

		f.workerPool.Submit(func() { // nolint: errcheck
			defer wg.Done()

			if err := f.updateRemoteURL(ctx, value, treeMutex, v4tree, v6tree); err != nil {
				cancel()
				f.logger.BindStr("url", value).WarningError("cannot update", err)

				select {
				case errorChan <- err:
				default:
				}
			}
		})
	}

	wg.Wait()

	select {
	case err := <-errorChan:
		return fmt.Errorf("cannot update trees: %w", err)
	default:
	}

	f.rwMutex.Lock()
	defer f.rwMutex.Unlock()

	f.treeV4 = v4tree
	f.treeV6 = v6tree

	return nil
}

func (f *Firehol) updateLocalFile(ctx context.Context, filename string,
	mutex sync.Locker,
	v4tree *bool_tree.TreeV4, v6tree *bool_tree.TreeV6) error {
	filefp, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}

	defer filefp.Close()

	return f.updateTrees(ctx, mutex, filefp, v4tree, v6tree)
}

func (f *Firehol) updateRemoteURL(ctx context.Context, url string,
	mutex sync.Locker,
	v4tree *bool_tree.TreeV4, v6tree *bool_tree.TreeV6) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("cannot build a request: %w", err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot request a remote URL %s: %w", url, err)
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body) // nolint: errcheck
		resp.Body.Close()
	}()

	return f.updateTrees(ctx, mutex, resp.Body, v4tree, v6tree)
}

func (f *Firehol) updateTrees(ctx context.Context,
	mutex sync.Locker,
	reader io.Reader,
	v4tree *bool_tree.TreeV4,
	v6tree *bool_tree.TreeV6) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		text := scanner.Text()
		text = fireholRegexpComment.ReplaceAllLiteralString(text, "")
		text = strings.TrimSpace(text)

		if text == "" {
			continue
		}

		ip, cidr, err := f.updateParseLine(text)
		if err != nil {
			return fmt.Errorf("cannot parse a line: %w", err)
		}

		if err := f.updateAddToTrees(ip, cidr, mutex, v4tree, v6tree); err != nil {
			return fmt.Errorf("cannot add a node to the tree: %w", err)
		}
	}

	if scanner.Err() != nil {
		return fmt.Errorf("cannot parse a response: %w", scanner.Err())
	}

	return nil
}

func (f *Firehol) updateParseLine(text string) (net.IP, uint, error) {
	_, ipnet, err := net.ParseCIDR(text)
	if err != nil {
		ipaddr := net.ParseIP(text)
		if ipaddr == nil {
			return nil, 0, fmt.Errorf("incorrect ip address %s", text)
		}

		ip4 := ipaddr.To4()
		if ip4 != nil {
			return ip4, fireholIPv4DefaultCIDR, nil
		}

		return ipaddr.To16(), fireholIPv6DefaultCIDR, nil
	}

	ones, _ := ipnet.Mask.Size()

	return ipnet.IP, uint(ones), nil
}

func (f *Firehol) updateAddToTrees(ip net.IP, cidr uint,
	mutex sync.Locker,
	v4tree *bool_tree.TreeV4, v6tree *bool_tree.TreeV6) error {
	mutex.Lock()
	defer mutex.Unlock()

	if ip.To4() != nil {
		addr := patricia.NewIPv4AddressFromBytes(ip, cidr)

		if _, _, err := v4tree.Set(addr, true); err != nil {
			return err // nolint: wrapcheck
		}
	} else {
		addr := patricia.NewIPv6Address(ip, cidr)

		if _, _, err := v6tree.Set(addr, true); err != nil {
			return err // nolint: wrapcheck
		}
	}

	return nil
}

func NewFirehol(logger mtglib.Logger, network mtglib.Network,
	downloadConcurrency uint,
	remoteURLs []string,
	localFiles []string) (*Firehol, error) {
	for _, v := range remoteURLs {
		parsed, err := url.Parse(v)
		if err != nil {
			return nil, fmt.Errorf("incorrect url %s: %w", v, err)
		}

		switch parsed.Scheme {
		case "http", "https":
		default:
			return nil, fmt.Errorf("unsupported url %s", v)
		}
	}

	for _, v := range localFiles {
		if stat, err := os.Stat(v); os.IsNotExist(err) || stat.IsDir() || stat.Mode().Perm()&0o400 == 0 {
			return nil, fmt.Errorf("%s is not a readable file", v)
		}
	}

	if downloadConcurrency == 0 {
		downloadConcurrency = 1
	}

	workerPool, _ := ants.NewPool(int(downloadConcurrency))
	ctx, cancel := context.WithCancel(context.Background())

	return &Firehol{
		ctx:        ctx,
		ctxCancel:  cancel,
		logger:     logger.Named("firehol"),
		httpClient: network.MakeHTTPClient(nil),
		treeV4:     bool_tree.NewTreeV4(),
		treeV6:     bool_tree.NewTreeV6(),
		workerPool: workerPool,
		remoteURLs: remoteURLs,
		localFiles: localFiles,
	}, nil
}
