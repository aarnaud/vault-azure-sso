package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gobuffalo/packr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
)

var cli = &cobra.Command{
	Run: func(cmd *cobra.Command, args []string) {
		startServer()
	},
}

var cliOptionVersion = &cobra.Command{
	Use:   "version",
	Short: "Print the version.",
	Long:  "The version of this program",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version 1.0.0")
	},
}

func init() {
	cli.AddCommand(cliOptionVersion)

	flags := cli.Flags()

	flags.BoolP("verbose", "v", false, "Enable verbose")
	if err := viper.BindPFlag("verbose", flags.Lookup("verbose")); err != nil {
		log.Fatal(err)
	}

	flags.Int("port", 3000, "HTTP port")
	if err := viper.BindPFlag("port", flags.Lookup("port")); err != nil {
		log.Fatal(err)
	}

	flags.String("vault-url", "http://127.0.0.1:8200", "Vault URL")
	if err := viper.BindPFlag("vault_url", flags.Lookup("vault-url")); err != nil {
		log.Fatal(err)
	}

	flags.String("client-id", "", "Application ID in App registrations")
	if err := viper.BindPFlag("client_id", flags.Lookup("client-id")); err != nil {
		log.Fatal(err)
	}

	flags.String("tenant-id", "", "Directory ID in Azure AD Properties")
	if err := viper.BindPFlag("tenant_id", flags.Lookup("tenant-id")); err != nil {
		log.Fatal(err)
	}
}

func main() {
	err := cli.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infoln(r.RemoteAddr, r.Method, r.URL, r.Referer())
		handler.ServeHTTP(w, r)
	})
}

func AuthCodeImplicitURL(c *oauth2.Config, nonce string) string {
	var buf bytes.Buffer
	buf.WriteString(c.Endpoint.AuthURL)
	v := url.Values{
		"response_type": {"id_token"},
		"client_id":     {c.ClientID},
		"response_mode": {"form_post"},
		"nonce":         {nonce},
	}
	if c.RedirectURL != "" {
		v.Set("redirect_uri", c.RedirectURL)
	}
	if len(c.Scopes) > 0 {
		v.Set("scope", strings.Join(c.Scopes, " "))
	}
	if strings.Contains(c.Endpoint.AuthURL, "?") {
		buf.WriteByte('&')
	} else {
		buf.WriteByte('?')
	}
	buf.WriteString(v.Encode())
	return buf.String()
}

func startServer() {
	viper.SetConfigName("azure-sso")            // name of config file (without extension)
	viper.AddConfigPath("/etc/vault/")          // path to look for the config file in
	viper.AddConfigPath("/etc/vault/azure-sso") // path to look for the config file in
	viper.AddConfigPath(".")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Infoln(err)
	}

	// EXPORT VAULT_AZURE_SSO_PORT=8000
	viper.SetEnvPrefix("VAULT_AZURE_SSO")
	viper.AutomaticEnv()

	if viper.GetBool("verbose") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	log.SetFormatter(&log.TextFormatter{})
	log.Info("Starting...")

	tpldir := packr.NewBox("./templates")

	tplFile, err := tpldir.FindString("index.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err := template.New("index").Parse(tplFile)
	if err != nil {
		log.Fatal(err)
	}

	AzureEndpoint := microsoft.AzureADEndpoint(viper.GetString("tenant_id"))
	oauthConfig := &oauth2.Config{
		ClientID:     viper.GetString("client_id"),
		ClientSecret: viper.GetString("client_secret"),
		Scopes:       []string{"openid", "offline_access", "email", "profile"},
		Endpoint:     AzureEndpoint,
	}

	oauthStateString := fmt.Sprintf("%d", rand.Int())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.FormValue("id_token")
		errorCode := r.FormValue("error")
		errorDescription := r.FormValue("error_description")

		data := map[string]interface{}{
			"VaultUrl":         viper.GetString("vault_url"),
			"AuthCodeURL":      AuthCodeImplicitURL(oauthConfig, oauthStateString),
			"AccessToken":      accessToken,
			"ErrorCode":        errorCode,
			"ErrorDescription": errorDescription,
		}
		//log.Debugf("Template Data: %+v", data)
		err := tmpl.Execute(w, data)
		if err != nil {
			log.Fatal(err)
		}
	})
	err = http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("port")), logRequest(http.DefaultServeMux))
	if err != nil {
		log.Fatal(err)
	}
}
