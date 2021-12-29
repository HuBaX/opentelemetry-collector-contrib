// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tailsamplingprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor"

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor/internal/sampling"
	"go.uber.org/zap"
)

func getNewAndPolicy(logger *zap.Logger, config AndCfg) (sampling.PolicyEvaluator, error) {
	subpolicies := []sampling.PolicyEvaluator{}
	ratelimiter := []sampling.PolicyEvaluator{}
	for _, subCfg := range config.SubPolicyCfg {
		evaluator, err := getEvaluator(logger, &subCfg)
		if err != nil {
			return nil, err
		}
		if subCfg.Type == RateLimiting {
			ratelimiter = append(ratelimiter, evaluator)
		} else {
			subpolicies = append(subpolicies, evaluator)
		}

	}
	//rate limiters will be evaluated last because of their stateful sampling logic
	subpolicies = append(subpolicies, ratelimiter...)
	return sampling.NewAndPolicy(logger, subpolicies), nil
}

// Return instance of composite sub-policy
func getEvaluator(logger *zap.Logger, cfg *SubPolicyCfg) (sampling.PolicyEvaluator, error) {
	switch cfg.Type {
	case AlwaysSample:
		return sampling.NewAlwaysSample(logger), nil
	case NumericAttribute:
		nafCfg := cfg.NumericAttributeCfg
		return sampling.NewNumericAttributeFilter(logger, nafCfg.Key, nafCfg.MinValue, nafCfg.MaxValue), nil
	case StringAttribute:
		safCfg := cfg.StringAttributeCfg
		return sampling.NewStringAttributeFilter(logger, safCfg.Key, safCfg.Values, safCfg.EnabledRegexMatching, safCfg.CacheMaxSize, safCfg.InvertMatch), nil
	case RateLimiting:
		rlfCfg := cfg.RateLimitingCfg
		return sampling.NewRateLimiting(logger, rlfCfg.SpansPerSecond), nil
	default:
		return nil, fmt.Errorf("unknown sampling sub-policy type %s", cfg.Type)
	}
}
