package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Ensure Provider satisfies various provider interfaces.
var _ provider.Provider = &GoogleSheetsProvider{}
var _ provider.ProviderWithFunctions = &GoogleSheetsProvider{}

// GoogleSheetsProvider defines the provider implementation.
type GoogleSheetsProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// GoogleSheetsProviderModel describes the provider data model.
type GoogleSheetsProviderModel struct {
	ServiceAccountKey types.String `tfsdk:"service_account_key"`
	Endpoint          types.String `tfsdk:"endpoint"`
}

func (p *GoogleSheetsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "gsheets"
	resp.Version = p.version
}

func (p *GoogleSheetsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"service_account_key": schema.StringAttribute{
				MarkdownDescription: "The Google Sheet ID",
				Optional:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The Google Sheet Endpoint, replace this to run tests with a mock server",
				Optional:            true,
			},
		},
	}
}

type CredentialsFile struct {
	Email string `json:"email"`
	// Type                    string   `json:"type"`
	// ProjectID               string   `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	// ClientEmail             string   `json:"client_email"`
	// ClientID                string   `json:"client_id"`
	// AuthURL                 string   `json:"auth_url"`
	TokenURL string `json:"token_url"`
	// AuthProviderX509CERTURL string   `json:"auth_provider_x509_cert_url"`
	// ClientX509CERTURL       string   `json:"client_x509_cert_url"`
	Scopes []string `json:"scopes"`
}

func (p *GoogleSheetsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data GoogleSheetsProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	opt := []option.ClientOption{}

	if !data.ServiceAccountKey.IsNull() {
		credentials := &CredentialsFile{}

		err := json.Unmarshal([]byte(data.ServiceAccountKey.ValueString()), credentials)
		if err != nil {

			resp.Diagnostics.AddError("unable to parse crentials as json, using file", err.Error())

			f, err := os.Open(data.ServiceAccountKey.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to open service account file",
					"",
				)
				return
			}
			defer f.Close()

			// Create a JWT configurations object for the Google service account
			err = json.NewDecoder(f).Decode(credentials)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to create service for google sheets",
					err.Error(),
				)
				return
			}
		}

		conf := &jwt.Config{
			Email:        credentials.Email,
			PrivateKey:   []byte(credentials.PrivateKey),
			PrivateKeyID: credentials.PrivateKeyID,
			TokenURL:     credentials.TokenURL,
			Scopes:       credentials.Scopes,
		}

		tflog.Info(ctx, fmt.Sprint(conf))

		client := conf.Client(ctx)
		opt = append(opt, option.WithHTTPClient(client))
	} else {
		opt = append(opt, option.WithoutAuthentication())

	}

	if !data.Endpoint.IsNull() {
		opt = append(opt, option.WithEndpoint(data.Endpoint.ValueString()))
	}

	gclient, err := sheets.NewService(ctx, opt...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create service for google sheets",
			err.Error(),
		)
		return
	}

	resp.DataSourceData = gclient
	resp.ResourceData = gclient
}

func (p *GoogleSheetsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSheetResource,
		NewRangeResource,
	}
}

func (p *GoogleSheetsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRowsDataSource,
	}
}

func (p *GoogleSheetsProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GoogleSheetsProvider{
			version: version,
		}
	}
}
