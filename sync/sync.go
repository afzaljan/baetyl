package sync

import (
	"encoding/json"
	"errors"

	"github.com/baetyl/baetyl-core/config"
	"github.com/baetyl/baetyl-core/event"
	"github.com/baetyl/baetyl-core/shadow"
	"github.com/baetyl/baetyl-go/http"
	"github.com/baetyl/baetyl-go/link"
	"github.com/baetyl/baetyl-go/log"
	"github.com/baetyl/baetyl-go/mqtt"
	bh "github.com/timshannon/bolthold"
	appv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

// ErrSyncTLSConfigMissing certificate bidirectional authentication is required for connection with cloud
var ErrSyncTLSConfigMissing = errors.New("Certificate bidirectional authentication is required for connection with cloud")

// Sync sync shadow and resources with cloud
type Sync struct {
	cent  *event.Center
	cfg   config.SyncConfig
	impl  appv1.DeploymentInterface
	http  *http.Client
	mqtt  *mqtt.Client
	store *bh.Store
	shad  *shadow.Shadow
	log   *log.Logger
}

// NewSync create a new sync
func NewSync(cfg config.SyncConfig, store *bh.Store, shad *shadow.Shadow, cent *event.Center) (*Sync, error) {
	ops, err := cfg.Cloud.HTTP.ToClientOptions()
	if err != nil {
		return nil, err
	}
	if ops.TLSConfig == nil {
		return nil, ErrSyncTLSConfigMissing
	}
	return &Sync{
		cfg:   cfg,
		cent:  cent,
		store: store,
		shad:  shad,
		http:  http.NewClient(ops),
		log:   log.With(log.Any("core", "sync")),
	}, nil
}

// Report reports info
func (s *Sync) Report(msg link.Message) error {
	data, err := s.http.PostJSON(s.cfg.Cloud.Report.URL, msg.Content)
	if err != nil {
		s.log.Error("failed to send report data", log.Error(err))
		return err
	}
	var res BackwardInfo
	err = json.Unmarshal(data, &res)
	if err != nil {
		s.log.Error("error to unmarshal response data returned", log.Error(err))
		return err
	}
	if res.Delta != nil {
		_, err = s.shad.Desire(res.Delta)
		if err != nil {
			return err
		}
	}
	return nil
}
