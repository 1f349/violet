package domains

import (
	"context"
	_ "embed"
	"github.com/1f349/violet/database"
	"github.com/1f349/violet/logger"
	"github.com/1f349/violet/utils"
	"strings"
	"time"
)

var Logger = logger.Logger.WithPrefix("Violet Domains")

// Domains is the domain list and management system.
type Domains struct {
	db *database.Queries
	c  *utils.CacheFlight[utils.StringKey, bool]
}

// New creates a new domain list
func New(db *database.Queries, ttl time.Duration) *Domains {
	return &Domains{
		db: db,
		c: utils.NewCacheFlight[utils.StringKey, bool](ttl, func(ctx context.Context, k utils.StringKey) (bool, error) {
			v, err := db.IsDomainActive(ctx, string(k))
			return v != 0, err
		}),
	}
}

// IsValid returns true if a domain is valid.
func (d *Domains) IsValid(ctx context.Context, host string) bool {
	domain, _, _ := utils.SplitDomainPort(host, 0)
	logger.Logger.Debug("Domain is valid", "domain", domain)

	// check root domains `www.example.com`, `example.com`, `com`
	for len(domain) > 0 {
		hasDomain, err := d.c.LoadOrStore(ctx, utils.StringKey(domain))
		if err != nil {
			Logger.Warn("Failed to get domain active state", "domain", domain, "err", err)
			// TODO: handle error
			return false
		}
		if hasDomain {
			return true
		}

		n := strings.IndexByte(domain, '.')
		if n == -1 {
			break
		}
		domain = domain[n+1:]
	}
	return false
}

func (d *Domains) Put(domain string, active bool) {
	err := d.db.AddDomain(context.Background(), database.AddDomainParams{
		Domain: domain,
		Active: active,
	})
	if err != nil {
		logger.Logger.Infof("Database error: %s\n", err)
	}
}

func (d *Domains) Delete(domain string) {
	err := d.db.DeleteDomain(context.Background(), domain)
	if err != nil {
		logger.Logger.Infof("Database error: %s\n", err)
	}
}
