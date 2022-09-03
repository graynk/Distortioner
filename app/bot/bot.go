package bot

import (
	"sync"

	"go.uber.org/zap"

	"github.com/graynk/distortioner/tools"
)

type distorterBot struct {
	adminID     int64
	rl          *tools.RateLimiter
	logger      *zap.SugaredLogger
	mu          *sync.Mutex
	graceWg     *sync.WaitGroup
	videoWorker *tools.VideoWorker
}

func NewDistorterBot(adminID int64, logger *zap.SugaredLogger) *distorterBot {
	return &distorterBot{
		adminID:     adminID,
		rl:          tools.NewRateLimiter(),
		logger:      logger,
		mu:          &sync.Mutex{},
		graceWg:     &sync.WaitGroup{},
		videoWorker: tools.NewVideoWorker(3),
	}
}

func (d distorterBot) Shutdown() {
	d.videoWorker.Shutdown()
	d.graceWg.Wait()
}
