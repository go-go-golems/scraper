package defaults

import (
	hackernews "github.com/go-go-golems/scraper/pkg/sites/hackernews"
	"github.com/go-go-golems/scraper/pkg/sites/jsdemo"
	"github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/go-go-golems/scraper/pkg/sites/slashdot"
)

func NewRegistry() (*registry.Registry, error) {
	ret := registry.New()
	if err := Register(ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func Register(r *registry.Registry) error {
	if r == nil {
		r = registry.New()
	}
	if err := hackernews.Register(r); err != nil {
		return err
	}
	if err := slashdot.Register(r); err != nil {
		return err
	}
	if err := jsdemo.Register(r); err != nil {
		return err
	}
	return nil
}
