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
	"github.com/kentik/patricia"
	"github.com/kentik/patricia/bool_tree"
	"github.com/panjf2000/ants/v2"
)

const (
	fireholIPv4DefaultCIDR = 32
	fireholIPv6DefaultCIDR = 128
)

var fireholRegexpComment = regexp.MustCompile(`\s*#.*?$`)

// Firehol is IPBlocklist which uses lists from FireHOL:
// https://iplists.firehol.org/
//
// It can use both local files and remote URLs. This is not necessary
// that blocklists should be taken from this website, we expect only
// compatible formats here.
//
// Example of the format:
//
//     # this is a comment
//     # to ignore
//     127.0.0.1   # you can specify an IP
//     10.0.0.0/8  # or cidr
type Firehol struct {
	ctx         context.Context
	ctxCancel   context.CancelFunc
	logger      mtglib.Logger
	updateMutex sync.RWMutex

	blocklists []files.File

	workerPool *ants.Pool
	treeV4     *bool_tree.TreeV4
	treeV6     *bool_tree.TreeV6
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

	if ip4 := ip.To4(); ip4 != nil {
		return f.containsIPv4(ip4)
	}

	return f.containsIPv6(ip.To16())
}

// Run starts a background update process.
//
// This is a blocking method so you probably want to run it in a
// goroutine.
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

func (f *Firehol) containsIPv4(addr net.IP) bool {
	ip := patricia.NewIPv4AddressFromBytes(addr, 32) // nolint: gomnd

	if ok, _ := f.treeV4.FindDeepestTag(ip); ok {
		return true
	}

	return false
}

func (f *Firehol) containsIPv6(addr net.IP) bool {
	ip := patricia.NewIPv6Address(addr, 128) // nolint: gomnd

	if ok, _ := f.treeV6.FindDeepestTag(ip); ok {
		return true
	}

	return false
}

func (f *Firehol) update() {
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()

	wg := &sync.WaitGroup{}
	wg.Add(len(f.blocklists))

	treeMutex := &sync.Mutex{}
	v4tree := bool_tree.NewTreeV4()
	v6tree := bool_tree.NewTreeV6()

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

			if err := f.updateFromFile(treeMutex, v4tree, v6tree, bufio.NewScanner(fileContent)); err != nil {
				logger.WarningError("update has failed", err)
			}
		}(v)
	}

	wg.Wait()

	f.updateMutex.Lock()
	defer f.updateMutex.Unlock()

	f.treeV4 = v4tree
	f.treeV6 = v6tree

	f.logger.Info("blocklist was updated")
}

func (f *Firehol) updateFromFile(mutex sync.Locker,
	v4tree *bool_tree.TreeV4,
	v6tree *bool_tree.TreeV6,
	scanner *bufio.Scanner) error {
	for scanner.Scan() {
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

		f.updateAddToTrees(ip, cidr, mutex, v4tree, v6tree)
	}

	if scanner.Err() != nil {
		return fmt.Errorf("cannot parse a file: %w", scanner.Err())
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
	v4tree *bool_tree.TreeV4, v6tree *bool_tree.TreeV6) {
	mutex.Lock()
	defer mutex.Unlock()

	if ip.To4() != nil {
		v4tree.Set(patricia.NewIPv4AddressFromBytes(ip, cidr), true)
	} else {
		v6tree.Set(patricia.NewIPv6Address(ip, cidr), true)
	}
}

// NewFirehol creates a new instance of FireHOL IP blocklist.
//
// This method does not start an update process so please execute Run
// when it is necessary.
func NewFirehol(logger mtglib.Logger, network mtglib.Network,
	downloadConcurrency uint,
	urls []string,
	localFiles []string) (*Firehol, error) {
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

	return NewFireholFromFiles(logger, downloadConcurrency, blocklists)
}

func NewFireholFromFiles(logger mtglib.Logger,
	downloadConcurrency uint,
	blocklists []files.File) (*Firehol, error) {
	if downloadConcurrency == 0 {
		downloadConcurrency = DefaultFireholDownloadConcurrency
	}

	workerPool, _ := ants.NewPool(int(downloadConcurrency))
	ctx, cancel := context.WithCancel(context.Background())

	return &Firehol{
		ctx:        ctx,
		ctxCancel:  cancel,
		logger:     logger.Named("firehol"),
		treeV4:     bool_tree.NewTreeV4(),
		treeV6:     bool_tree.NewTreeV6(),
		workerPool: workerPool,
		blocklists: blocklists,
	}, nil
}
