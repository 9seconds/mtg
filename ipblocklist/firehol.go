package ipblocklist

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/ipblocklist/files"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/panjf2000/ants/v2"
	"github.com/yl2chen/cidranger"
)

var (
	fireholRegexpComment = regexp.MustCompile(`\s*#.*?$`)

	fireholIPv4DefaultCIDR = net.CIDRMask(32, 32)   //nolint: gomnd
	fireholIPv6DefaultCIDR = net.CIDRMask(128, 128) //nolint: gomnd
)

// FireholUpdateCallback defines a signature of the callback that has to be
// execute when ip list is updated.
type FireholUpdateCallback func(context.Context, int)

// Firehol is [mtglib.IPBlocklist] which uses lists from FireHOL:
// https://iplists.firehol.org/
//
// It can use both local files and remote URLs. This is not necessary that
// blocklists should be taken from this website, we expect only compatible
// formats here.
//
// Example of the format:
//
//	# this is a comment
//	# to ignore
//	127.0.0.1   # you can specify an IP
//	10.0.0.0/8  # or cidr
type Firehol struct {
	ctx         context.Context
	ctxCancel   context.CancelFunc
	logger      mtglib.Logger
	updateMutex sync.RWMutex

	updateCallback FireholUpdateCallback
	ranger         cidranger.Ranger

	blocklists []files.File

	workerPool *ants.Pool
}

// Shutdown stop a background update process.
func (f *Firehol) Shutdown() {
	f.ctxCancel()
}

// Contains is given IP list can be found in FireHOL blocklists.
func (f *Firehol) Contains(ip net.IP) bool {
	if ip == nil {
		return true
	}

	f.updateMutex.RLock()
	defer f.updateMutex.RUnlock()

	ok, err := f.ranger.Contains(ip)
	if err != nil {
		f.logger.BindStr("ip", ip.String()).DebugError("Cannot check if ip is present", err)
	}

	return ok && err == nil
}

// Run starts a background update process.
//
// This is a blocking method so you probably want to run it in a goroutine.
func (f *Firehol) Run(updateEach time.Duration) {
	if updateEach == 0 {
		updateEach = DefaultFireholUpdateEach
	}

	ticker := time.NewTicker(updateEach)

	defer func() {
		ticker.Stop()

		select {
		case <-ticker.C:
		default:
		}
	}()

	f.update()

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			f.update()
		}
	}
}

func (f *Firehol) update() {
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(len(f.blocklists))

	mutex := &sync.Mutex{}
	ranger := cidranger.NewPCTrieRanger()

	for _, v := range f.blocklists {
		go func(file files.File) {
			defer wg.Done()

			logger := f.logger.BindStr("filename", file.String())

			fileContent, err := file.Open(ctx)
			if err != nil {
				logger.WarningError("update has failed", err)

				return
			}

			defer fileContent.Close()

			if err := f.updateFromFile(mutex, ranger, bufio.NewScanner(fileContent)); err != nil {
				logger.WarningError("update has failed", err)
			}
		}(v)
	}

	wg.Wait()

	f.updateMutex.Lock()
	defer f.updateMutex.Unlock()

	f.ranger = ranger

	if f.updateCallback != nil {
		f.updateCallback(ctx, ranger.Len())
	}

	f.logger.Info("ip list was updated")
}

func (f *Firehol) updateFromFile(mutex sync.Locker,
	ranger cidranger.Ranger,
	scanner *bufio.Scanner,
) error {
	for scanner.Scan() {
		text := scanner.Text()
		text = fireholRegexpComment.ReplaceAllLiteralString(text, "")
		text = strings.TrimSpace(text)

		if text == "" {
			continue
		}

		ipnet, err := f.updateParseLine(text)
		if err != nil {
			return fmt.Errorf("cannot parse a line: %w", err)
		}

		mutex.Lock()
		err = ranger.Insert(cidranger.NewBasicRangerEntry(*ipnet))
		mutex.Unlock()

		if err != nil {
			return fmt.Errorf("cannot insert %v into ranger: %w", ipnet, err)
		}
	}

	if scanner.Err() != nil {
		return fmt.Errorf("cannot parse a file: %w", scanner.Err())
	}

	return nil
}

func (f *Firehol) updateParseLine(text string) (*net.IPNet, error) {
	if _, ipnet, err := net.ParseCIDR(text); err == nil {
		return ipnet, nil
	}

	ipaddr := net.ParseIP(text)
	if ipaddr == nil {
		return nil, fmt.Errorf("incorrect ip address %s", text)
	}

	mask := fireholIPv4DefaultCIDR

	if ipaddr.To4() == nil {
		mask = fireholIPv6DefaultCIDR
	}

	return &net.IPNet{
		IP:   ipaddr,
		Mask: mask,
	}, nil
}

// NewFirehol creates a new instance of FireHOL IP blocklist.
//
// This method does not start an update process so please execute Run when it
// is necessary.
func NewFirehol(logger mtglib.Logger, network mtglib.Network,
	downloadConcurrency uint,
	urls []string,
	localFiles []string,
	updateCallback FireholUpdateCallback,
) (*Firehol, error) {
	blocklists := []files.File{}

	for _, v := range localFiles {
		file, err := files.NewLocal(v)
		if err != nil {
			return nil, fmt.Errorf("cannot create a local file %s: %w", v, err)
		}

		blocklists = append(blocklists, file)
	}

	httpClient := network.MakeHTTPClient(nil)

	for _, v := range urls {
		file, err := files.NewHTTP(httpClient, v)
		if err != nil {
			return nil, fmt.Errorf("cannot create a HTTP file %s: %w", v, err)
		}

		blocklists = append(blocklists, file)
	}

	return NewFireholFromFiles(logger, downloadConcurrency, blocklists, updateCallback)
}

// NewFirehol creates a new instance of FireHOL IP blocklist.
//
// This method creates this instances from a given list of files.
func NewFireholFromFiles(logger mtglib.Logger,
	downloadConcurrency uint,
	blocklists []files.File,
	updateCallback FireholUpdateCallback,
) (*Firehol, error) {
	if downloadConcurrency == 0 {
		downloadConcurrency = DefaultFireholDownloadConcurrency
	}

	workerPool, _ := ants.NewPool(int(downloadConcurrency))
	ctx, cancel := context.WithCancel(context.Background())

	return &Firehol{
		ctx:            ctx,
		ctxCancel:      cancel,
		logger:         logger.Named("firehol"),
		ranger:         cidranger.NewPCTrieRanger(),
		workerPool:     workerPool,
		blocklists:     blocklists,
		updateCallback: updateCallback,
	}, nil
}
