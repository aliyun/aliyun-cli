package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/aliyun-cli/resource"
	"github.com/aliyun/aliyun-cli/config"
	"net"
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
	"strings"
)

func NewResolveCommand() (*cli.Command) {
	cmd := &cli.Command{
		Name: "resolve",
		Usage: "resolve <productCode>",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) != 1 {
				cli.Errorf("Error: invalid args %v", args)
				ctx.Command().PrintUsage()
			} else {
				ResolveEndpoint(args[0])
			}
			return nil
		},
	}
	return cmd
}

func ResolveEndpoint(code string) {
	profile, err := config.LoadCurrentProfile()
	if err != nil {
		cli.Errorf("Error: please configure first")
		return
	}

	regions, err := config.GetRegions(&profile)
	if err != nil {
		cli.Errorf("Error: get region failed %s, please configure first", err)
		return
	}

	client, err := profile.GetClient()
	if err != nil {
		cli.Errorf("Error: get client failed %s, please configure first", err)
		return
	}

	library := meta.LoadLibrary(resource.NewReader())
	product, ok := library.GetProduct(code)
	if !ok {
		suggestions := GetProductSuggestions(library, code)
		msg := ""
		if len(suggestions) > 0 {
			for i, s := range suggestions {
				if i == 0 {
					msg = "did you mean: " + s
				} else {
					msg = msg + " or " + s
				}
			}
		}
		cli.Errorf("Error: unknown product %s ", code)
		cli.Warningf(" %s \n", msg)
		return
	}

	fmt.Printf("\nProduct: %s (%s)\n", product.Code, product.Name[i18n.GetLanguage()])
	fmt.Printf("Version: %s \n", product.Version)
	fmt.Printf("Link: %s\n", product.GetDocumentLink(i18n.GetLanguage()))

	fmt.Printf("\nLocationServiceCode: %s \n", product.LocationServiceCode)

	for _, pattern := range product.EndpointPatterns {
		fmt.Printf("- EndpointPattern: %s\n", pattern)
	}

	fmt.Printf("\nTest endpoints...:\n")


	regionalPattern := ""
	globalEndpoints := make(map[string]bool, 0)
	regionalEndpoints := make(map[string]string, 0)
	for _, region := range regions {
		ep, lep := product.TryGetEndpoints(string(region.RegionId), client)
		fmt.Printf("- %s(%s): ", region.RegionId, region.LocalName)

		known := false
		if ep == "" {
			cli.Warning("EP=,")
		} else {
			if ValidateDomain(ep) {
				known = true
				regionalEndpoints[region.RegionId] = ep
				cli.Noticef("EP=%s, ", ep)
			} else {
				cli.Errorf("EP=%s(BAD), ", ep)
			}
		}

		if lep == "" {
			cli.Warning("LC=")
		} else {
			if ValidateDomain(lep) {
				known = true
				delete(regionalEndpoints, region.RegionId)
				cli.Noticef("LC=%s", lep)
			} else {
				cli.Errorf("LC=%s(BAD)", lep)
			}
		}

		if !known {
			for _, pattern := range product.EndpointPatterns {
				if strings.Contains(pattern, "[RegionId]") {
					ep = strings.Replace(pattern, "[RegionId]", region.RegionId, 1)
					if ValidateDomain(ep) {
						regionalEndpoints[region.RegionId] = ep
						if regionalPattern == "" {
							regionalPattern = pattern
						} else if regionalPattern != pattern {
							regionalPattern = "Conflict!!!"
						}
						cli.Noticef(", GUESS=%s", ep)
					}
				} else {
					if ValidateDomain(pattern) {
						globalEndpoints[pattern] = true
					} else {
						globalEndpoints[pattern] = false
					}
				}
			}
		}
		fmt.Println()
	}
	for k, v := range globalEndpoints {
		if v {
			cli.Noticef("- GLOBAL: %s\n", k)
		} else {
			cli.Errorf("- GLOBAL: %s(BAD)\n", k)
		}
	}

	if len(regionalEndpoints) == 0 {
		return
	}

	fmt.Printf("\n  endpoint_pattern: %s", regionalPattern)
	fmt.Printf("\n  regional_endpoints:\n")
	for k, v := range regionalEndpoints {
		fmt.Printf("    %s: %s\n", k, v)
	}
}

func ValidateDomain(domain string) bool {
	_, err := net.ResolveIPAddr("", domain)
	return err == nil
}
