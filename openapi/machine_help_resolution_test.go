package openapi

import (
	"errors"
	"os"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/stretchr/testify/require"
)

type machineHelpResolutionRepository struct {
	machineHelpRepository
	reads  []string
	apiErr error
}

func (r *machineHelpResolutionRepository) GetProducts() (*canonicalmeta.ProductsIndex, error) {
	r.reads = append(r.reads, "products")
	return r.machineHelpRepository.GetProducts()
}

func (r *machineHelpResolutionRepository) GetVersionIndex(product, version string) (*canonicalmeta.VersionIndex, error) {
	r.reads = append(r.reads, "index/"+product+"/"+version)
	return r.machineHelpRepository.GetVersionIndex(product, version)
}

func (r *machineHelpResolutionRepository) GetAPI(product, version, name string) (*canonicalmeta.API, error) {
	r.reads = append(r.reads, "api/"+product+"/"+version+"/"+name)
	if r.apiErr != nil {
		return nil, r.apiErr
	}
	return r.machineHelpRepository.GetAPI(product, version, name)
}

func TestMachineHelpCommandResolutionObservesOnlyLoadedAPIs(t *testing.T) {
	readFailure := errors.New("API metadata unavailable")
	for _, test := range []struct {
		name, operation, command, version string
		apiErr                            error
		wantResolved, wantError           bool
		wantReads                         []string
	}{
		{
			name: "action in explicitly selected version", operation: "action", command: "create-report", version: "2026-01-01",
			wantResolved: true,
			wantReads:    []string{"products", "index/demo/2026-01-01", "api/demo/2026-01-01/CreateReport"},
		},
		{
			name: "response Help", operation: "response", command: "create-report", version: "2026-01-01",
			wantResolved: true,
			wantReads:    []string{"products", "index/demo/2026-01-01", "api/demo/2026-01-01/CreateReport"},
		},
		{
			name: "invalid parameter after API resolution", operation: "parameter", command: "create-report", version: "2026-01-01",
			wantResolved: true, wantError: true,
			wantReads: []string{"products", "index/demo/2026-01-01", "api/demo/2026-01-01/CreateReport"},
		},
		{
			name: "product Help enumerates without selecting an API", operation: "product", version: "2026-01-01",
			wantReads: []string{"products", "index/demo/2026-01-01"},
		},
		{
			name: "command absent from default version", operation: "action", command: "create-report",
			wantError: true,
			wantReads: []string{"products", "index/demo/2025-01-01"},
		},
		{
			name: "unknown command", operation: "action", command: "private-customer-input", version: "2026-01-01",
			wantError: true,
			wantReads: []string{"products", "index/demo/2026-01-01"},
		},
		{
			name: "API metadata read failure", operation: "action", command: "create-report", version: "2026-01-01", apiErr: readFailure,
			wantError: true,
			wantReads: []string{"products", "index/demo/2026-01-01", "api/demo/2026-01-01/CreateReport"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			build := func(observer bool) (any, error) {
				repository := &machineHelpResolutionRepository{
					machineHelpRepository: canonicalmeta.NewRepository(os.DirFS("../canonicalmeta/testdata")),
					apiErr:                test.apiErr,
				}
				service := newMachineHelpService(repository)
				var resolved [][]string
				if observer {
					service.onCommandResolved = func(product, command string) {
						resolved = append(resolved, []string{product, command})
						require.Equal(t, test.wantReads, repository.reads, "notify only after loading the selected API")
					}
				}
				var document any
				var err error
				switch test.operation {
				case "product":
					document, err = service.buildProductForStyle("DEMO", test.version, "kebab")
				case "response":
					document, err = service.buildAPIResponse("DEMO", test.command, test.version)
				case "parameter":
					var action *machineHelpAPIDocument
					action, err = service.buildAPI("DEMO", test.command, test.version)
					if err == nil {
						document, err = buildParameterHelpDocument(action, "--private-unknown-parameter")
					}
				default:
					document, err = service.buildAPI("DEMO", test.command, test.version)
				}
				require.Equal(t, test.wantReads, repository.reads, "the observer adds no metadata reads")
				if observer && test.wantResolved {
					require.Equal(t, [][]string{{"DEMO", test.command}}, resolved)
				} else {
					require.Empty(t, resolved)
				}
				return document, err
			}

			withObserver, err := build(true)
			withoutObserver, withoutErr := build(false)
			require.Equal(t, test.wantError, err != nil)
			require.Equal(t, test.wantError, withoutErr != nil)
			if test.apiErr != nil {
				require.ErrorIs(t, err, test.apiErr)
				require.ErrorIs(t, withoutErr, test.apiErr)
			}
			require.Equal(t, withoutObserver, withObserver, "optional observer preserves the Help document")
		})
	}
}
