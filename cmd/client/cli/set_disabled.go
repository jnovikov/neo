package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"neo/internal/client"

	neopb "neo/lib/genproto/neo"
)

type setDisabledCli struct {
	*baseCLI
	exploitID string
	disabled  bool
}

func NewSetDisabled(_ *cobra.Command, args []string, cfg *client.Config, disabled bool) NeoCLI {
	return &setDisabledCli{
		baseCLI:   &baseCLI{cfg},
		exploitID: args[0],
		disabled:  disabled,
	}
}

func (sc *setDisabledCli) Run(ctx context.Context) error {
	c, err := sc.client()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	state, err := c.Ping(ctx, neopb.PingRequest_CONFIG_REQUEST)
	if err != nil {
		return fmt.Errorf("failed to get config from server: %w", err)
	}

	var spl *neopb.ExploitState
	for _, v := range state.GetExploits() {
		if v.GetExploitId() == sc.exploitID {
			spl = v
			break
		}
	}

	if spl == nil {
		return fmt.Errorf("exploit %s does not exist", sc.exploitID)
	}
	if err := c.SetExploitDisabled(ctx, spl.GetExploitId(), sc.disabled); err != nil {
		return fmt.Errorf("set disabled failed: %w", err)
	}

	return nil
}
