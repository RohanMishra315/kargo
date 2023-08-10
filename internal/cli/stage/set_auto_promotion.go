package stage

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/utils/pointer"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

func newEnableAutoPromotion(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable-auto-promotion",
		Args:    cobra.ExactArgs(2),
		Example: "kargo stage enable-auto-promotion (PROJECT) (STAGE)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := strings.TrimSpace(args[0])
			if project == "" {
				return errors.New("project is required")
			}
			stage := strings.TrimSpace(args[1])
			if stage == "" {
				return errors.New("stage is required")
			}
			return setAutoPromotionForStage(ctx, opt, project, stage, true)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}

func newDisableAutoPromotion(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "disable-auto-promotion",
		Args:    cobra.ExactArgs(2),
		Example: "kargo stage disable-auto-promotion (PROJECT) (STAGE)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := strings.TrimSpace(args[0])
			if project == "" {
				return errors.New("project is required")
			}
			stage := strings.TrimSpace(args[1])
			if stage == "" {
				return errors.New("stage is required")
			}
			return setAutoPromotionForStage(ctx, opt, project, stage, false)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}

func setAutoPromotionForStage(ctx context.Context, opt *option.Option, project, stage string, enable bool) error {
	client := svcv1alpha1connect.NewKargoServiceClient(http.DefaultClient, opt.ServerURL, opt.ClientOption)
	resp, err := client.SetAutoPromotionForStage(ctx,
		connect.NewRequest(&v1alpha1.SetAutoPromotionForStageRequest{
			Project: project,
			Stage:   stage,
			Enable:  enable,
		}))
	if err != nil {
		return errors.Wrapf(err, "set auto promotion for stage: %q", stage)
	}
	if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
		res := "Disabled"
		if enable {
			res = "Enabled"
		}
		_, _ = fmt.Fprintf(opt.IOStreams.Out,
			"%s AutoPromotion for Stage %q", res, resp.Msg.GetPromotionPolicy().GetStage())
		return nil
	}
	printer, err := opt.PrintFlags.ToPrinter()
	if err != nil {
		return errors.Wrap(err, "new printer")
	}
	policy := typesv1alpha1.FromPromotionPolicyProto(resp.Msg.GetPromotionPolicy())
	_ = printer.PrintObj(policy, opt.IOStreams.Out)
	return nil
}