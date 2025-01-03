package directives

import (
	"context"
	"fmt"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/freight"
	intyaml "github.com/akuity/kargo/internal/yaml"
)

func init() {
	builtins.RegisterPromotionStepRunner(
		newHelmImageUpdater(),
		&StepRunnerPermissions{
			AllowKargoClient: true,
		},
	)
}

// helmImageUpdater is an implementation of the PromotionStepRunner interface
// that updates image references in a Helm values file.
//
// TODO(krancour): Remove for v1.3.0.
type helmImageUpdater struct {
	schemaLoader gojsonschema.JSONLoader
}

// newHelmImageUpdater returns an implementation of the PromotionStepRunner
// interface that updates image references in a Helm values file.
func newHelmImageUpdater() PromotionStepRunner {
	r := &helmImageUpdater{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner HealthCheckStepRunner interface.
func (h *helmImageUpdater) Name() string {
	return "helm-update-image"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (h *helmImageUpdater) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	failure := PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}

	if err := h.validate(stepCtx.Config); err != nil {
		return failure, err
	}

	// Convert the configuration into a typed struct
	cfg, err := ConfigToStruct[HelmUpdateImageConfig](stepCtx.Config)
	if err != nil {
		return failure, fmt.Errorf("could not convert config into %s config: %w", h.Name(), err)
	}

	return h.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates helmImageUpdater configuration against a JSON schema.
func (h *helmImageUpdater) validate(cfg Config) error {
	return validate(h.schemaLoader, gojsonschema.NewGoLoader(cfg), h.Name())
}

func (h *helmImageUpdater) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg HelmUpdateImageConfig,
) (PromotionStepResult, error) {
	updates, fullImageRefs, err := h.generateImageUpdates(ctx, stepCtx, cfg)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("failed to generate image updates: %w", err)
	}

	result := PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}
	if len(updates) > 0 {
		if err = h.updateValuesFile(stepCtx.WorkDir, cfg.Path, updates); err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("values file update failed: %w", err)
		}

		if commitMsg := h.generateCommitMessage(cfg.Path, fullImageRefs); commitMsg != "" {
			result.Output = map[string]any{
				"commitMessage": commitMsg,
			}
		}
	}
	return result, nil
}

func (h *helmImageUpdater) generateImageUpdates(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg HelmUpdateImageConfig,
) ([]intyaml.Update, []string, error) {
	updates := make([]intyaml.Update, len(cfg.Images))
	fullImageRefs := make([]string, 0, len(cfg.Images))

	for i, image := range cfg.Images {
		desiredOrigin := h.getDesiredOrigin(image.FromOrigin)

		targetImage, err := freight.FindImage(
			ctx,
			stepCtx.KargoClient,
			stepCtx.Project,
			stepCtx.FreightRequests,
			desiredOrigin,
			stepCtx.Freight.References(),
			image.Image,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find image %s: %w", image.Image, err)
		}

		value, imageRef, err := h.getImageValues(targetImage, image.Value)
		if err != nil {
			return nil, nil, err
		}

		updates[i] = intyaml.Update{
			Key:   image.Key,
			Value: value,
		}
		fullImageRefs = append(fullImageRefs, imageRef)
	}
	return updates, fullImageRefs, nil
}

func (h *helmImageUpdater) getDesiredOrigin(fromOrigin *ChartFromOrigin) *kargoapi.FreightOrigin {
	if fromOrigin == nil {
		return nil
	}
	return &kargoapi.FreightOrigin{
		Kind: kargoapi.FreightOriginKind(fromOrigin.Kind),
		Name: fromOrigin.Name,
	}
}

func (h *helmImageUpdater) getImageValues(image *kargoapi.Image, valueType string) (string, string, error) {
	switch valueType {
	case ImageAndTag:
		imageRef := fmt.Sprintf("%s:%s", image.RepoURL, image.Tag)
		return imageRef, imageRef, nil
	case Tag:
		return image.Tag, fmt.Sprintf("%s:%s", image.RepoURL, image.Tag), nil
	case ImageAndDigest:
		imageRef := fmt.Sprintf("%s@%s", image.RepoURL, image.Digest)
		return imageRef, imageRef, nil
	case Digest:
		return image.Digest, fmt.Sprintf("%s@%s", image.RepoURL, image.Digest), nil
	default:
		return "", "", fmt.Errorf("unknown image value type %q", valueType)
	}
}

func (h *helmImageUpdater) updateValuesFile(workDir string, path string, updates []intyaml.Update) error {
	absValuesFile, err := securejoin.SecureJoin(workDir, path)
	if err != nil {
		return fmt.Errorf("error joining path %q: %w", path, err)
	}
	if err := intyaml.SetStringsInFile(absValuesFile, updates); err != nil {
		return fmt.Errorf("error updating image references in values file %q: %w", path, err)
	}
	return nil
}

func (h *helmImageUpdater) generateCommitMessage(path string, fullImageRefs []string) string {
	if len(fullImageRefs) == 0 {
		return ""
	}

	var commitMsg strings.Builder
	_, _ = commitMsg.WriteString(fmt.Sprintf("Updated %s to use new image", path))
	if len(fullImageRefs) > 1 {
		_, _ = commitMsg.WriteString("s")
	}
	_, _ = commitMsg.WriteString("\n")

	for _, s := range fullImageRefs {
		_, _ = commitMsg.WriteString(fmt.Sprintf("\n- %s", s))
	}

	return commitMsg.String()
}
