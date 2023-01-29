package responsetransformer

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

type Rules []Rule

type Rule struct {
	Prefix   string            `yaml:"prefix"`
	Path     string            `yaml:"path"`
	Method   string            `yaml:"method"`
	Default  string            `yaml:"default"`
	Override string            `yaml:"override"`
	Query    map[string]string `yaml:"query"`
	Headers  map[string]string `yaml:"headers"`
}

func (r Rules) Apply(res *http.Response) error {
	// only apply rules for successes
	if res.StatusCode != http.StatusOK {
		return nil
	}
	// if rule found, apply to response
	if rule := r.find(res); rule != nil {
		applyRuleToResponse(*rule, res)
	}
	return nil
}

func applyRuleToResponse(rule Rule, res *http.Response) {
	if rule.Override != "" {
		log.Trace().Msg("Overriding Cache-Control header")
		res.Header.Set("Cache-Control", rule.Override)
	} else if rule.Default != "" && res.Header.Get("Cache-Control") == "" {
		log.Trace().Msg("Applying default Cache-Control header")
		res.Header.Set("Cache-Control", rule.Default)
	}
	for name, value := range rule.Headers {
		log.Trace().Msgf("Setting header %s", name)
		res.Header.Set(name, value)
	}
}

func (r Rules) find(res *http.Response) *Rule {
	log.Trace().Msgf("Finding rule for request %s:%s", res.Request.Method, res.Request.URL.Path)
rulesLoop:
	for _, rule := range r {
		log.Trace().Msgf("Checking rule %+v", rule)
		if rule.Method == "" && res.Request.Method != http.MethodGet {
			continue
		}
		if rule.Method != "" && rule.Method != res.Request.Method {
			continue
		}
		if rule.Path != "" && rule.Path != res.Request.URL.Path {
			continue
		}
		if rule.Prefix != "" && !strings.HasPrefix(res.Request.URL.Path, rule.Prefix) {
			continue
		}
		if len(rule.Query) > 0 {
			qry := res.Request.URL.Query()
			for name, value := range rule.Query {
				if value == "" && !qry.Has(name) {
					continue rulesLoop
				} else if value != "" && qry.Get(name) != value {
					continue rulesLoop
				}
			}
		}
		// disable unsafe (un-GET) methods for now, they aren't working
		if rule.Method != "" {
			log.Warn().Msg("Non-GET method rules not supported")
			continue
		}
		return &rule
	}
	return nil
}
