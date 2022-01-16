// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tailsamplingprocessor

import (
	"reflect"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor/internal/sampling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config"
	"go.uber.org/zap"
)

func TestGetNewAndPolicy(t *testing.T) {
	assert := assert.New(t)
	cfg := &Config{
		ProcessorSettings:       config.NewProcessorSettings(config.NewComponentID(typeStr)),
		DecisionWait:            10 * time.Second,
		NumTraces:               100,
		ExpectedNewTracesPerSec: 10,
		PolicyCfgs: []PolicyCfg{
			{
				Name: "and-policy",
				Type: And,
				AndCfg: AndCfg{
					SubPolicyCfg: []PolicyCfg{
						{
							Name:                "policy-1",
							Type:                NumericAttribute,
							NumericAttributeCfg: NumericAttributeCfg{Key: "number", MinValue: 50, MaxValue: 100},
						},
						{
							Name:            "policy-2",
							Type:            RateLimiting,
							RateLimitingCfg: RateLimitingCfg{SpansPerSecond: 10},
						},
						{
							Name:               "policy-3",
							Type:               StringAttribute,
							StringAttributeCfg: StringAttributeCfg{Key: "string", Values: []string{"value1", "value2"}},
						},
						{
							Name:          "test-policy-5",
							Type:          StatusCode,
							StatusCodeCfg: StatusCodeCfg{StatusCodes: []string{"ERROR", "UNSET"}},
						},
						{
							Name:       "test-policy-2",
							Type:       Latency,
							LatencyCfg: LatencyCfg{ThresholdMs: 5000},
						},
						{
							Name: "test-policy-1",
							Type: AlwaysSample,
						},
						{
							Name:             "test-policy-4",
							Type:             Probabilistic,
							ProbabilisticCfg: ProbabilisticCfg{HashSalt: "custom-salt", SamplingPercentage: 0.1},
						},
					},
				},
			},
		},
	}
	andCfg := cfg.PolicyCfgs[0].AndCfg
	logger := zap.NewNop()
	numeric := sampling.NewNumericAttributeFilter(logger, "number", 50, 100)
	ratelimit := sampling.NewRateLimiting(logger, 10)
	str := sampling.NewStringAttributeFilter(logger, "string", []string{"value1, value2"}, false, -1, false)
	status, _ := sampling.NewStatusCodeFilter(logger, []string{"ERROR", "UNSET"})
	lat := sampling.NewLatency(logger, 5000)
	always := sampling.NewAlwaysSample(logger)
	prob := sampling.NewProbabilisticSampler(logger, "custom-salt", 0.1)

	and, err := getNewAndPolicy(logger, andCfg)

	require.NotNil(t, and)
	assert.NoError(err)
	assert.NotNil(cfg.ProcessorSettings)
	assert.Equal(10*time.Second, cfg.DecisionWait)
	assert.Equal(uint64(100), cfg.NumTraces)
	assert.Equal(uint64(10), cfg.ExpectedNewTracesPerSec)
	assert.IsType(&sampling.And{}, and)
	samplingAnd := and.(*sampling.And)
	//Checking if the rate limiter will be executed last, since it is the only stateful policy to date
	reflect.DeepEqual([]sampling.PolicyEvaluator{numeric, str, ratelimit, status, lat, always, prob}, samplingAnd.SubPolicies)
}
