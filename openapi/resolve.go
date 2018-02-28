package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/aliyun-cli/resource"
	"github.com/aliyun/aliyun-cli/config"
	"net"
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
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
		cli.Errorf("Error: unknown product %s\n", code)
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

	for _, region := range regions {
		ep, lep := product.TryGetEndpoints(string(region.RegionId), client)
		fmt.Printf("- %s(%s): ", region.RegionId, region.LocalName)
		if ep == "" {
			cli.Warning("EP=,")
		} else {
			if ValidateDomain(ep) {
				cli.Noticef("EP=%s, ", ep)
			} else {
				cli.Errorf("EP=%s(BAD), ", ep)
			}
		}

		if lep == "" {
			cli.Warning("LC=\n")
		} else {
			if ValidateDomain(lep) {
				cli.Noticef("LC=%s\n", lep)
			} else {
				cli.Errorf("LC=%s(BAD)\n", lep)
			}
		}
	}
}

func ValidateDomain(domain string) bool {
	_, err := net.ResolveIPAddr("", domain)
	return err == nil
}
