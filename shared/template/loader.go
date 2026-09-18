package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	texttemplate "text/template"

	"github.com/navire-dev/navire/shared/models"
	sharedvalidation "github.com/navire-dev/navire/shared/validation"
	"gopkg.in/yaml.v3"
)

// YAMLFiles returns all YAML files below dir in deterministic path order.
func YAMLFiles(dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("stat templates directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("templates path %s is not a directory", dir)
	}

	files := make([]string, 0)
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan templates: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func isYAMLFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}

func Parse(data []byte, filename string) (*Definition, error) {
	var definition Definition
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&definition); err != nil {
		return nil, fmt.Errorf("parse template %s: %w", filename, err)
	}
	definition.Rules = normalizeRules(definition.Rules)
	return &definition, nil
}

func (d *Definition) Validate() error {
	if !sharedvalidation.IsKey(d.Key) {
		return fmt.Errorf("template key must contain 1 to %d lowercase letters, digits, '-' or '_' and start with a letter or digit", sharedvalidation.MaxKeyLength)
	}
	if len(d.Variants) == 0 {
		return fmt.Errorf("at least one variant is required")
	}
	if len(d.Providers.Entries) == 0 {
		return fmt.Errorf("at least one provider is required")
	}
	if _, err := d.ResolvePolicy("", ""); err != nil {
		return fmt.Errorf("defaults: %w", err)
	}
	if err := d.validateVariants(); err != nil {
		return err
	}
	if err := d.validateProviders(); err != nil {
		return err
	}
	return d.validateRules()
}

func (d *Definition) validateVariants() error {
	statuses := make(map[string]string)
	for name, variant := range d.Variants {
		fields, err := validateVariant(name, variant)
		if err != nil {
			return err
		}
		variant.RequiredFields = fields
		d.Variants[name] = variant
		status := strings.ToLower(strings.TrimSpace(variant.Status))
		if status == "" {
			continue
		}
		if otherName, exists := statuses[status]; exists {
			return fmt.Errorf("variants %q and %q have duplicate status %q", name, otherName, variant.Status)
		}
		statuses[status] = name
	}
	return nil
}

func validateVariant(name string, variant Variant) ([]string, error) {
	if name == "" {
		return nil, fmt.Errorf("variant name is required")
	}
	if variant.Title == "" {
		return nil, fmt.Errorf("variant %q title is required", name)
	}
	if variant.Body == "" {
		return nil, fmt.Errorf("variant %q body is required", name)
	}
	if _, err := models.ParseEventState(string(variant.State)); err != nil {
		return nil, fmt.Errorf("variant %q: %w", name, err)
	}
	if _, err := ParsePriority(string(variant.Priority)); err != nil {
		return nil, fmt.Errorf("variant %q: %w", name, err)
	}
	fields, err := fieldsForVariant(name, variant)
	if err != nil {
		return nil, err
	}
	return fields, nil
}

func (d *Definition) validateProviders() error {
	for providerName, provider := range d.Providers.Entries {
		if len(provider.Endpoints) == 0 {
			return fmt.Errorf("provider %q must define at least one endpoint", providerName)
		}
		for endpointName := range provider.Endpoints {
			if _, err := d.ResolvePolicy(providerName, endpointName); err != nil {
				return fmt.Errorf("provider %q endpoint %q: %w", providerName, endpointName, err)
			}
		}
	}
	return nil
}

func (d *Definition) validateRules() error {
	for index, rule := range d.Rules {
		if err := validateRuleCondition(rule.Condition); err != nil {
			return fmt.Errorf("rules[%d]: %w", index, err)
		}
		if len(rule.Providers) == 0 {
			return fmt.Errorf("rules[%d]: at least one provider destination is required", index)
		}
		for providerName, destination := range rule.Providers {
			provider, ok := d.Providers.Entries[providerName]
			if !ok {
				return fmt.Errorf("rules[%d]: provider %q is not defined", index, providerName)
			}
			if len(destination.Endpoints) == 0 {
				return fmt.Errorf("rules[%d]: provider %q must define at least one endpoint", index, providerName)
			}
			for _, endpointName := range destination.Endpoints {
				if _, ok := provider.Endpoints[endpointName]; !ok {
					return fmt.Errorf("rules[%d]: endpoint %q is not defined for provider %q", index, endpointName, providerName)
				}
			}
		}
	}
	return nil
}

func validateRuleCondition(condition RuleCondition) error {
	if _, err := ParsePriority(string(condition.Priority)); err != nil {
		return fmt.Errorf("condition: %w", err)
	}
	if condition.State != "" {
		if _, err := models.ParseEventState(string(condition.State)); err != nil {
			return fmt.Errorf("condition: %w", err)
		}
	}
	for key := range condition.Data {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("condition data contains an empty key")
		}
	}
	return nil
}

func normalizeRules(rules []RoutingRule) []RoutingRule {
	if len(rules) < 2 {
		return rules
	}
	result := make([]RoutingRule, 0, len(rules))
	indexes := make(map[string]int, len(rules))
	for _, rule := range rules {
		keyBytes, err := json.Marshal(rule.Condition)
		if err != nil {
			result = append(result, rule)
			continue
		}
		key := string(keyBytes)
		index, exists := indexes[key]
		if !exists {
			indexes[key] = len(result)
			result = append(result, cloneRule(rule))
			continue
		}
		mergeRuleProviders(&result[index], rule)
	}
	return result
}

func cloneRule(rule RoutingRule) RoutingRule {
	cloned := RoutingRule{
		Condition: rule.Condition,
		Providers: make(map[string]RuleProvider, len(rule.Providers)),
	}
	for providerName, destination := range rule.Providers {
		cloned.Providers[providerName] = RuleProvider{
			Endpoints: append([]string(nil), destination.Endpoints...),
		}
	}
	return cloned
}

func mergeRuleProviders(target *RoutingRule, source RoutingRule) {
	if target.Providers == nil {
		target.Providers = make(map[string]RuleProvider)
	}
	for providerName, sourceDestination := range source.Providers {
		targetDestination := target.Providers[providerName]
		for _, endpointName := range sourceDestination.Endpoints {
			if !contains(targetDestination.Endpoints, endpointName) {
				targetDestination.Endpoints = append(targetDestination.Endpoints, endpointName)
			}
		}
		target.Providers[providerName] = targetDestination
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// ResolveVariant returns an explicitly requested variant or selects one from
// data.status when the producer does not provide a variant.
func (d *Definition) ResolveVariant(name string, data map[string]any) (string, Variant, error) {
	if name != "" {
		variant, ok := d.Variants[name]
		if !ok {
			return "", Variant{}, fmt.Errorf("variant %q not found", name)
		}
		return name, variant, nil
	}
	status, _ := data["status"].(string)
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return "", Variant{}, fmt.Errorf("variant is required when data.status is absent")
	}
	for variantName, variant := range d.Variants {
		if strings.ToLower(strings.TrimSpace(variant.Status)) == status {
			return variantName, variant, nil
		}
	}
	return "", Variant{}, fmt.Errorf("no variant matches status %q", status)
}

// ResolvePriority applies request override, variant, template default, then
// the Navire default, in that order.
func (d *Definition) ResolvePriority(variantName, requestOverride string) (Priority, error) {
	variant, ok := d.Variants[variantName]
	if !ok {
		return "", fmt.Errorf("variant %q not found", variantName)
	}
	if requestOverride != "" {
		return ParsePriority(requestOverride)
	}
	if variant.Priority != "" {
		return ParsePriority(string(variant.Priority))
	}
	return PriorityNormal, nil
}

func DefaultDeliveryPolicy() DeliveryPolicy {
	return DeliveryPolicy{
		Timeout: "10s",
		Retry: RetryPolicy{
			MaxAttempts: 1,
			Backoff:     models.BackoffNone,
		},
	}
}

func (d *Definition) ResolvePolicy(providerName, endpointName string) (DeliveryPolicy, error) {
	policy := DefaultDeliveryPolicy()
	policy = mergePolicy(policy, d.Providers.Defaults)

	if providerName != "" || endpointName != "" {
		provider, ok := d.Providers.Entries[providerName]
		if !ok {
			return DeliveryPolicy{}, fmt.Errorf("provider %q is not defined", providerName)
		}
		override, ok := provider.Endpoints[endpointName]
		if !ok {
			return DeliveryPolicy{}, fmt.Errorf("endpoint %q is not defined for provider %q", endpointName, providerName)
		}
		policy = mergePolicyOverride(policy, override)
	}
	if err := policy.Validate(); err != nil {
		return DeliveryPolicy{}, err
	}
	return policy, nil
}

func mergePolicy(base, override DeliveryPolicy) DeliveryPolicy {
	if override.Timeout != "" {
		base.Timeout = override.Timeout
	}
	if override.Retry.MaxAttempts != 0 {
		base.Retry.MaxAttempts = override.Retry.MaxAttempts
	}
	if override.Retry.Backoff != "" {
		base.Retry.Backoff = override.Retry.Backoff
	}
	if override.Retry.InitialWait != "" {
		base.Retry.InitialWait = override.Retry.InitialWait
	}
	return base
}

func mergePolicyOverride(base DeliveryPolicy, override ProviderOverride) DeliveryPolicy {
	if override.Timeout != nil {
		base.Timeout = *override.Timeout
	}
	if override.Retry.MaxAttempts != nil {
		base.Retry.MaxAttempts = *override.Retry.MaxAttempts
	}
	if override.Retry.Backoff != nil {
		base.Retry.Backoff = *override.Retry.Backoff
	}
	if override.Retry.InitialWait != nil {
		base.Retry.InitialWait = *override.Retry.InitialWait
	}
	return base
}

func (d *Definition) ProviderNames() []string {
	names := make([]string, 0, len(d.Providers.Entries))
	for name := range d.Providers.Entries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (d *Definition) EndpointNames(providerName string) []string {
	provider, ok := d.Providers.Entries[providerName]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(provider.Endpoints))
	for name := range provider.Endpoints {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func compile(name, source string) (*texttemplate.Template, error) {
	tmpl, err := texttemplate.New(name).Option("missingkey=error").Parse(source)
	if err != nil {
		return nil, fmt.Errorf("compile %s: %w", name, err)
	}
	return tmpl, nil
}
