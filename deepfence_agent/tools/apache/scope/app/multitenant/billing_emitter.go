package multitenant

import (
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"math"
	"strings"
	"sync"
	"time"

	"context"
	log "github.com/sirupsen/logrus"
	billing "github.com/weaveworks/billing-client"

	"github.com/weaveworks/scope/app"
	"github.com/weaveworks/scope/report"
)

// BillingEmitterConfig has everything we need to make a billing emitter
type BillingEmitterConfig struct {
	Enabled         bool
	DefaultInterval time.Duration
	UserIDer        UserIDer
}

// RegisterFlags registers the billing emitter flags with the main flag set.
func (cfg *BillingEmitterConfig) RegisterFlags(f *flag.FlagSet) {
	f.BoolVar(&cfg.Enabled, "app.billing.enabled", false, "enable emitting billing info")
	f.DurationVar(&cfg.DefaultInterval, "app.billing.default-publish-interval", 3*time.Second, "default publish interval to assume for reports")
}

// BillingEmitter is the billing emitter
type BillingEmitter struct {
	app.Collector
	BillingEmitterConfig
	billing *billing.Client

	sync.Mutex
	intervalCache map[string]time.Duration
	rounding      map[string]float64
}

// NewBillingEmitter changes a new billing emitter which emits billing events
func NewBillingEmitter(upstream app.Collector, billingClient *billing.Client, cfg BillingEmitterConfig) (*BillingEmitter, error) {
	return &BillingEmitter{
		Collector:            upstream,
		billing:              billingClient,
		BillingEmitterConfig: cfg,
		intervalCache:        make(map[string]time.Duration),
		rounding:             make(map[string]float64),
	}, nil
}

// Add implements app.Collector
func (e *BillingEmitter) Add(ctx context.Context, rep report.Report, buf []byte) error {
	now := time.Now().UTC()
	userID, err := e.UserIDer(ctx)
	if err != nil {
		// Underlying collector needs to get userID too, so it's OK to abort
		// here. If this fails, so will underlying collector so no point
		// proceeding.
		return err
	}
	rowKey, colKey := calculateDynamoKeys(userID, now)

	interval := e.reportInterval(rep)
	// Cache the last-known value of interval for this user, and use
	// it if we didn't find one in this report.
	e.Lock()
	if interval != 0 {
		e.intervalCache[userID] = interval
	} else {
		if lastKnown, found := e.intervalCache[userID]; found {
			interval = lastKnown
		} else {
			interval = e.DefaultInterval
		}
	}
	// Billing takes an integer number of seconds, so keep track of the amount lost to rounding
	nodeSeconds := interval.Seconds()*float64(len(rep.Host.Nodes)) + e.rounding[userID]
	rounding := nodeSeconds - math.Floor(nodeSeconds)
	e.rounding[userID] = rounding
	e.Unlock()

	hasher := sha256.New()
	hasher.Write(buf)
	hash := "sha256:" + base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	weaveNetCount := 0
	if hasWeaveNet(rep) {
		weaveNetCount = 1
	}

	amounts := billing.Amounts{
		billing.ContainerSeconds: int64(interval/time.Second) * int64(len(rep.Container.Nodes)),
		billing.NodeSeconds:      int64(nodeSeconds),
		billing.WeaveNetSeconds:  int64(interval/time.Second) * int64(weaveNetCount),
	}
	metadata := map[string]string{
		"row_key": rowKey,
		"col_key": colKey,
	}

	err = e.billing.AddAmounts(
		hash,
		userID,
		now,
		amounts,
		metadata,
	)
	if err != nil {
		// No return, because we want to proceed even if we fail to emit
		// billing data, so that defects in the billing system don't break
		// report collection. Just log the fact & carry on.
		log.Errorf("Failed emitting billing data: %v", err)
	}

	return e.Collector.Add(ctx, rep, buf)
}

// reportInterval tries to find the custom report interval of this report. If
// it is malformed, or not set, it returns zero.
func (e *BillingEmitter) reportInterval(r report.Report) time.Duration {
	if r.Window != 0 {
		return r.Window
	}
	var inter string
	for _, c := range r.Process.Nodes {
		cmd, ok := c.Latest.Lookup("cmdline")
		if !ok {
			continue
		}
		if strings.Contains(cmd, "deepfence-discovery") &&
			strings.Contains(cmd, "probe.publish.interval") {
			cmds := strings.SplitAfter(cmd, "probe.publish.interval")
			aft := strings.Split(cmds[1], " ")
			if aft[0] == "" {
				inter = aft[1]
			} else {
				inter = aft[0][1:]
			}

		}
	}
	if inter == "" {
		return 0
	}
	d, err := time.ParseDuration(inter)
	if err != nil {
		return 0
	}
	return d
}

// Tries to determine if this report came from a host running Weave Net
func hasWeaveNet(r report.Report) bool {
	for _, n := range r.Overlay.Nodes {
		overlayType, _ := report.ParseOverlayNodeID(n.ID)
		if overlayType == report.WeaveOverlayPeerPrefix {
			return true
		}
	}

	return false
}

// Close shuts down the billing emitter and billing client flushing events.
func (e *BillingEmitter) Close() {
	e.Collector.Close()
	_ = e.billing.Close()
}
