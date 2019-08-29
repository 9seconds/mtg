package newcli

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/9seconds/mtg/newconfig"
	"github.com/9seconds/mtg/newstats"
	"github.com/9seconds/mtg/ntp"
)

func Proxy() error {
	atom := zap.NewAtomicLevel()
	switch {
	case newconfig.C.Debug:
		atom.SetLevel(zapcore.DebugLevel)
	case newconfig.C.Verbose:
		atom.SetLevel(zapcore.InfoLevel)
	default:
		atom.SetLevel(zapcore.ErrorLevel)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stderr),
		atom,
	))
	zap.ReplaceGlobals(logger)
	defer logger.Sync() // nolint: errcheck

	if err := newconfig.InitPublicAddress(); err != nil {
		Fatal(err.Error())
	}
	zap.S().Debugw("Configuration", "config", newconfig.C)

	if len(newconfig.C.AdTag) > 0 {
		zap.S().Infow("Use middle proxy connection to Telegram")
		diff, err := ntp.Fetch()
		if err != nil {
			Fatal("Cannot fetch time data from NTP")
		}
		if diff > time.Second {
			Fatal("Your local time is skewed and drift is bigger than a second. Please sync your time.")
		}
		go ntp.AutoUpdate()
	} else {
		zap.S().Infow("Use direct connection to Telegram")
	}

	PrintJSONStdout(newconfig.GetURLs())

	if err := newstats.Init(); err != nil {
		Fatal(err.Error())
	}

	return nil
}
