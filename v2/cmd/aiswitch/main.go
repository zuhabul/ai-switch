package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zuhabul/ai-switch/v2/internal/adapter"
	"github.com/zuhabul/ai-switch/v2/internal/model"
	"github.com/zuhabul/ai-switch/v2/internal/service"
	"github.com/zuhabul/ai-switch/v2/internal/store"
	"github.com/zuhabul/ai-switch/v2/internal/vault"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	ctx := context.Background()
	svc := service.New(store.NewFileStore(defaultStatePath()))
	if err := svc.Init(ctx); err != nil {
		exitErr(err)
	}
	v := vault.NewFileVault(defaultVaultPath())

	switch os.Args[1] {
	case "init":
		fmt.Printf("initialized state at %s\n", defaultStatePath())
	case "adapters":
		printJSON(adapter.NewRegistry().List())
	case "profile":
		handleProfile(ctx, svc, os.Args[2:])
	case "policy":
		handlePolicy(ctx, svc, os.Args[2:])
	case "health":
		handleHealth(ctx, svc, os.Args[2:])
	case "route":
		handleRoute(ctx, svc, os.Args[2:])
	case "lease":
		handleLease(ctx, svc, os.Args[2:])
	case "secret":
		handleSecret(ctx, svc, v, os.Args[2:])
	case "runtime":
		handleRuntime(ctx, svc, v, os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func handleProfile(ctx context.Context, svc *service.Service, args []string) {
	if len(args) == 0 {
		exitErr(fmt.Errorf("profile requires subcommand add|list|cooldown"))
	}
	switch args[0] {
	case "add":
		fs := flag.NewFlagSet("profile add", flag.ExitOnError)
		id := fs.String("id", "", "profile id")
		provider := fs.String("provider", "", "provider")
		frontend := fs.String("frontend", "", "frontend")
		auth := fs.String("auth", "", "auth method")
		protocol := fs.String("protocol", "", "protocol")
		account := fs.String("account", "", "account label")
		priority := fs.Int("priority", 0, "priority")
		tags := fs.String("tags", "", "comma-separated tags")
		budget := fs.Float64("budget", 0, "daily budget USD")
		enabled := fs.Bool("enabled", true, "enabled")
		_ = fs.Parse(args[1:])
		p := model.Profile{
			ID:             *id,
			Provider:       *provider,
			Frontend:       *frontend,
			AuthMethod:     *auth,
			Protocol:       *protocol,
			Account:        *account,
			Priority:       *priority,
			Enabled:        *enabled,
			Tags:           splitCSV(*tags),
			BudgetDailyUSD: *budget,
		}
		if err := svc.AddProfile(ctx, p); err != nil {
			exitErr(err)
		}
		fmt.Println("ok")
	case "list":
		ps, err := svc.ListProfiles(ctx)
		if err != nil {
			exitErr(err)
		}
		printJSON(ps)
	case "cooldown":
		fs := flag.NewFlagSet("profile cooldown", flag.ExitOnError)
		id := fs.String("id", "", "profile id")
		duration := fs.Duration("for", 15*time.Minute, "cooldown duration")
		_ = fs.Parse(args[1:])
		if err := svc.SetCooldown(ctx, *id, *duration); err != nil {
			exitErr(err)
		}
		fmt.Println("ok")
	default:
		exitErr(fmt.Errorf("unknown profile command %s", args[0]))
	}
}

func handlePolicy(ctx context.Context, svc *service.Service, args []string) {
	if len(args) == 0 {
		exitErr(fmt.Errorf("policy requires subcommand add|list"))
	}
	switch args[0] {
	case "add":
		fs := flag.NewFlagSet("policy add", flag.ExitOnError)
		name := fs.String("name", "", "rule name")
		priority := fs.Int("priority", 100, "priority")
		frontends := fs.String("frontends", "", "comma-separated frontends")
		tasks := fs.String("tasks", "", "comma-separated task classes")
		allowProviders := fs.String("allow-providers", "", "comma-separated allowed providers")
		denyProviders := fs.String("deny-providers", "", "comma-separated denied providers")
		requireTags := fs.String("require-any-tag", "", "comma-separated required tags")
		requireAuth := fs.String("require-auth", "", "comma-separated required auth methods")
		maxBudget := fs.Float64("max-budget", 0, "max budget usd")
		_ = fs.Parse(args[1:])
		rule := model.PolicyRule{
			Name:               *name,
			Priority:           *priority,
			Frontends:          splitCSV(*frontends),
			TaskClasses:        splitCSV(*tasks),
			AllowProviders:     splitCSV(*allowProviders),
			DenyProviders:      splitCSV(*denyProviders),
			RequireAnyTag:      splitCSV(*requireTags),
			RequireAuthMethods: splitCSV(*requireAuth),
			MaxBudgetDailyUSD:  *maxBudget,
		}
		if err := svc.AddPolicy(ctx, rule); err != nil {
			exitErr(err)
		}
		fmt.Println("ok")
	case "list":
		policies, err := svc.ListPolicies(ctx)
		if err != nil {
			exitErr(err)
		}
		printJSON(policies)
	default:
		exitErr(fmt.Errorf("unknown policy command %s", args[0]))
	}
}

func handleHealth(ctx context.Context, svc *service.Service, args []string) {
	if len(args) == 0 {
		exitErr(fmt.Errorf("health requires subcommand set"))
	}
	if args[0] != "set" {
		exitErr(fmt.Errorf("unknown health command %s", args[0]))
	}
	fs := flag.NewFlagSet("health set", flag.ExitOnError)
	id := fs.String("id", "", "profile id")
	r5 := fs.Int("r5m", 0, "remaining requests in 5 min")
	rh := fs.Int("rh", 0, "remaining requests hour")
	lat := fs.Int("latency", 0, "estimated latency ms")
	errRate := fs.Float64("error", 0, "recent error rate percent")
	_ = fs.Parse(args[1:])
	hs := model.HealthSnapshot{
		ProfileID:              *id,
		RemainingRequests5Min:  *r5,
		RemainingRequestsHour:  *rh,
		EstimatedLatencyMS:     *lat,
		RecentErrorRatePercent: *errRate,
		UpdatedAt:              time.Now().UTC(),
	}
	if err := svc.UpdateHealth(ctx, hs); err != nil {
		exitErr(err)
	}
	fmt.Println("ok")
}

func handleRoute(ctx context.Context, svc *service.Service, args []string) {
	fs := flag.NewFlagSet("route", flag.ExitOnError)
	frontend := fs.String("frontend", "", "frontend")
	task := fs.String("task", "coding", "task class")
	protocol := fs.String("protocol", "", "required protocol")
	providers := fs.String("providers", "", "comma-separated preferred providers")
	tags := fs.String("tags", "", "comma-separated required tags")
	owner := fs.String("owner", "", "owner")
	_ = fs.Parse(args)

	d, err := svc.Route(ctx, model.TaskRequest{
		Frontend:           *frontend,
		TaskClass:          *task,
		RequiredProtocol:   *protocol,
		PreferredProviders: splitCSV(*providers),
		RequireTags:        splitCSV(*tags),
		Owner:              *owner,
	})
	if err != nil {
		exitErr(fmt.Errorf("%w; decision=%s", err, asJSON(d)))
	}
	printJSON(d)
}

func handleLease(ctx context.Context, svc *service.Service, args []string) {
	if len(args) == 0 {
		exitErr(fmt.Errorf("lease requires subcommand acquire|release|list"))
	}
	switch args[0] {
	case "acquire":
		fs := flag.NewFlagSet("lease acquire", flag.ExitOnError)
		profileID := fs.String("profile", "", "profile id")
		frontend := fs.String("frontend", "", "frontend")
		owner := fs.String("owner", "", "owner")
		ttl := fs.Duration("ttl", 15*time.Minute, "ttl")
		_ = fs.Parse(args[1:])
		lease, err := svc.AcquireLease(ctx, *profileID, *frontend, *owner, *ttl)
		if err != nil {
			exitErr(err)
		}
		printJSON(lease)
	case "release":
		fs := flag.NewFlagSet("lease release", flag.ExitOnError)
		id := fs.String("id", "", "lease id")
		_ = fs.Parse(args[1:])
		if err := svc.ReleaseLease(ctx, *id); err != nil {
			exitErr(err)
		}
		fmt.Println("ok")
	case "list":
		leases, err := svc.ListLeases(ctx)
		if err != nil {
			exitErr(err)
		}
		printJSON(leases)
	default:
		exitErr(fmt.Errorf("unknown lease command %s", args[0]))
	}
}

func handleSecret(ctx context.Context, svc *service.Service, v *vault.FileVault, args []string) {
	if len(args) == 0 {
		exitErr(fmt.Errorf("secret requires subcommand set|get|list|delete|bind|unbind|bindings"))
	}
	switch args[0] {
	case "set":
		fs := flag.NewFlagSet("secret set", flag.ExitOnError)
		name := fs.String("name", "", "secret name")
		value := fs.String("value", "", "secret value")
		_ = fs.Parse(args[1:])
		if err := v.Set(*name, *value); err != nil {
			exitErr(err)
		}
		fmt.Println("ok")
	case "get":
		fs := flag.NewFlagSet("secret get", flag.ExitOnError)
		name := fs.String("name", "", "secret name")
		_ = fs.Parse(args[1:])
		val, err := v.Get(*name)
		if err != nil {
			exitErr(err)
		}
		fmt.Println(val)
	case "list":
		names, err := v.List()
		if err != nil {
			exitErr(err)
		}
		printJSON(names)
	case "delete":
		fs := flag.NewFlagSet("secret delete", flag.ExitOnError)
		name := fs.String("name", "", "secret name")
		_ = fs.Parse(args[1:])
		if err := v.Delete(*name); err != nil {
			exitErr(err)
		}
		fmt.Println("ok")
	case "bind":
		fs := flag.NewFlagSet("secret bind", flag.ExitOnError)
		profileID := fs.String("profile", "", "profile id")
		envVar := fs.String("env", "", "env var")
		secretName := fs.String("name", "", "secret name")
		_ = fs.Parse(args[1:])
		if _, err := v.Get(*secretName); err != nil {
			exitErr(fmt.Errorf("secret validation failed: %w", err))
		}
		if err := svc.BindSecret(ctx, *profileID, *envVar, *secretName); err != nil {
			exitErr(err)
		}
		fmt.Println("ok")
	case "unbind":
		fs := flag.NewFlagSet("secret unbind", flag.ExitOnError)
		profileID := fs.String("profile", "", "profile id")
		envVar := fs.String("env", "", "env var")
		_ = fs.Parse(args[1:])
		if err := svc.UnbindSecret(ctx, *profileID, *envVar); err != nil {
			exitErr(err)
		}
		fmt.Println("ok")
	case "bindings":
		fs := flag.NewFlagSet("secret bindings", flag.ExitOnError)
		profileID := fs.String("profile", "", "profile id")
		_ = fs.Parse(args[1:])
		bindings, err := svc.ListSecretBindings(ctx, *profileID)
		if err != nil {
			exitErr(err)
		}
		printJSON(bindings)
	default:
		exitErr(fmt.Errorf("unknown secret command %s", args[0]))
	}
}

func handleRuntime(ctx context.Context, svc *service.Service, v *vault.FileVault, args []string) {
	if len(args) == 0 {
		exitErr(fmt.Errorf("runtime requires subcommand plan|release"))
	}
	switch args[0] {
	case "plan":
		fs := flag.NewFlagSet("runtime plan", flag.ExitOnError)
		frontend := fs.String("frontend", "", "frontend")
		task := fs.String("task", "coding", "task class")
		protocol := fs.String("protocol", "", "required protocol")
		providers := fs.String("providers", "", "comma-separated preferred providers")
		tags := fs.String("tags", "", "comma-separated required tags")
		owner := fs.String("owner", "runtime-plan-cli", "owner")
		cwd := fs.String("cwd", "", "working directory")
		modelName := fs.String("model", "", "model")
		prompt := fs.String("prompt", "", "prompt")
		leaseTTL := fs.Duration("lease-ttl", 15*time.Minute, "lease ttl")
		_ = fs.Parse(args[1:])
		decision, err := svc.Route(ctx, model.TaskRequest{
			Frontend:           *frontend,
			TaskClass:          *task,
			RequiredProtocol:   *protocol,
			PreferredProviders: splitCSV(*providers),
			RequireTags:        splitCSV(*tags),
			Owner:              *owner,
		})
		if err != nil {
			exitErr(fmt.Errorf("%w; decision=%s", err, asJSON(decision)))
		}
		profile, err := svc.GetProfile(ctx, decision.ProfileID)
		if err != nil {
			exitErr(err)
		}
		lease, err := svc.AcquireLease(ctx, decision.ProfileID, profile.Frontend, *owner, *leaseTTL)
		if err != nil {
			exitErr(err)
		}
		releaseOnErr := true
		defer func() {
			if releaseOnErr {
				_ = svc.ReleaseLease(ctx, lease.ID)
			}
		}()
		bindings, err := svc.ListSecretBindings(ctx, decision.ProfileID)
		if err != nil {
			exitErr(err)
		}
		env := map[string]string{
			"AI_SWITCH_PROFILE_ID": decision.ProfileID,
			"AI_SWITCH_LEASE_ID":   lease.ID,
		}
		for envVar, secretKey := range bindings {
			val, err := v.Get(secretKey)
			if err != nil {
				exitErr(err)
			}
			env[envVar] = val
		}
		spec, err := adapter.BuildDefault(profile.Frontend, adapter.LaunchRequest{
			Frontend: *frontend,
			Prompt:   *prompt,
			Cwd:      *cwd,
			Model:    *modelName,
			Args:     fs.Args(),
		})
		if err != nil {
			exitErr(err)
		}
		for k, val := range spec.Env {
			env[k] = val
		}
		releaseOnErr = false
		printJSON(model.RuntimePlan{
			ProfileID: decision.ProfileID,
			LeaseID:   lease.ID,
			Command:   spec.Command,
			Args:      spec.Args,
			Env:       env,
			Reasons:   decision.Reasons,
		})
	case "release":
		fs := flag.NewFlagSet("runtime release", flag.ExitOnError)
		leaseID := fs.String("lease", "", "lease id")
		_ = fs.Parse(args[1:])
		if err := svc.ReleaseLease(ctx, *leaseID); err != nil {
			exitErr(err)
		}
		fmt.Println("ok")
	default:
		exitErr(fmt.Errorf("unknown runtime command %s", args[0]))
	}
}

func splitCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func asJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func defaultStatePath() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ".aiswitch/state.json"
	}
	return filepath.Join(h, ".config", "ai-switch-v2", "state.json")
}

func defaultVaultPath() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ".aiswitch/secrets.enc.json"
	}
	return filepath.Join(h, ".config", "ai-switch-v2", "secrets.enc.json")
}

func usage() {
	fmt.Println(`aiswitch v2

Usage:
  aiswitch init
  aiswitch adapters
  aiswitch profile add --id ID --provider openai --frontend codex --auth chatgpt --protocol app_server [--priority 10]
  aiswitch profile list
  aiswitch profile cooldown --id ID --for 30m
  aiswitch policy add --name NAME [--frontends codex] [--allow-providers openai,google]
  aiswitch policy list
  aiswitch health set --id ID --r5m 30 --rh 600 --latency 220 --error 0.1
  aiswitch route --frontend codex --task coding --protocol app_server
  aiswitch lease acquire --profile ID --frontend codex --owner multica --ttl 15m
  aiswitch lease release --id LEASE_ID
  aiswitch lease list
  aiswitch secret set --name KEY --value VALUE
  aiswitch secret get --name KEY
  aiswitch secret list
  aiswitch secret delete --name KEY
  aiswitch secret bind --profile PROFILE --env ENV_VAR --name SECRET_KEY
  aiswitch secret bindings --profile PROFILE
  aiswitch runtime plan --frontend codex --task coding --protocol app_server [--lease-ttl 15m]
  aiswitch runtime release --lease LEASE_ID`)
}
