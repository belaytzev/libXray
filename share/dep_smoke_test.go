package share

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xtls/xray-core/infra/conf"
)

func TestQuicParamsConfigAccessible(t *testing.T) {
	// Smoke test: verify the new QuicParamsConfig struct from xray-core v26.3.27 is accessible
	qp := &conf.QuicParamsConfig{
		Congestion: "brutal",
		BrutalUp:   conf.Bandwidth("100 mbps"),
		BrutalDown: conf.Bandwidth("200 mbps"),
	}
	assert.Equal(t, "brutal", qp.Congestion)
	assert.Equal(t, conf.Bandwidth("100 mbps"), qp.BrutalUp)
	assert.Equal(t, conf.Bandwidth("200 mbps"), qp.BrutalDown)

	// Verify FinalMask can hold QuicParams
	fm := &conf.FinalMask{
		QuicParams: qp,
	}
	assert.NotNil(t, fm.QuicParams)
	assert.Equal(t, "brutal", fm.QuicParams.Congestion)
}
