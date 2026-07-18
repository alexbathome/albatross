package albatross

import (
	"flag"
	"fmt"
	"os"
)

// FlagSet wraps the standard library's flag.FlagSet with two extras: flags
// may be bound to an environment variable that overrides the parsed argument
// value, and flags may be marked required, which Parse enforces.
type FlagSet struct {
	*flag.FlagSet

	specs []flagSpec
}

type flagSpec struct {
	name     string
	env      string
	required bool
}

// FlagFunc is the shape of the flag definition methods on flag.FlagSet
// (String, Int, Bool, Duration, ...).
type FlagFunc[T any] func(name string, value T, usage string) *T

// NewFlagSet returns a FlagSet with the given name.
func NewFlagSet(name string) *FlagSet {
	// ContinueOnError so that required-flag and env-var failures from our
	// Parse surface the same way as the standard library's parse errors.
	return &FlagSet{FlagSet: flag.NewFlagSet(name, flag.ContinueOnError)}
}

// Flag defines a flag via fn, which must be a definition method of the
// wrapped FlagSet (fs.String, fs.Int, fs.Bool, ...). It returns a getter
// for the flag's value; call it after Parse. If env is non-empty and that
// environment variable is set, it overrides whatever the arguments
// provided; the value is converted with the same parsing rules as the flag
// itself. Required flags must be provided by argument or environment —
// never via the default — or Parse fails.
func Flag[T any](fs *FlagSet, fn FlagFunc[T], name string, value T, usage, env string, required bool) func() T {
	if env != "" {
		usage = fmt.Sprintf("%s (env %s)", usage, env)
	}
	fs.specs = append(fs.specs, flagSpec{name: name, env: env, required: required})
	p := fn(name, value, usage)
	return func() T { return *p }
}

// Parse parses args, applies environment overrides, and verifies required
// flags were provided.
func (fs *FlagSet) Parse(args []string) error {
	if err := fs.FlagSet.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	provided := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { provided[f.Name] = true })

	for _, s := range fs.specs {
		if s.env == "" {
			continue
		}
		v, ok := os.LookupEnv(s.env)
		if !ok {
			continue
		}
		if err := fs.Set(s.name, v); err != nil {
			return fmt.Errorf("invalid value %q in env var %s for flag -%s: %w", v, s.env, s.name, err)
		}
		provided[s.name] = true
	}

	for _, s := range fs.specs {
		if !s.required || provided[s.name] {
			continue
		}
		if s.env != "" {
			return fmt.Errorf("required flag -%s (or env var %s) was not set", s.name, s.env)
		}
		return fmt.Errorf("required flag -%s was not set", s.name)
	}
	return nil
}
